package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type (
	GeneralSection struct {
		LocationId     string `toml:"location_id"`
		LocationPretty string `toml:"location_pretty"`
		Domain         string `toml:"domain"`
	}
	HttpSection struct {
		Destinations []string `toml:"destinations"`
	}
	Root struct {
		General GeneralSection
		Http    HttpSection
	}
)

func Unmarshal() (Root, error) {
	content, err := ioutil.ReadFile("./config.toml")
	if err != nil {
		log.Fatal(err)
	}

	var (
		v Root
	)
	err = toml.Unmarshal(content, &v)

	return v, err
}

func isIpv6(ip string) bool {
	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return false
		case ':':
			return true
		}
	}
	return false
}

func ByteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

//go:embed views/*
var viewsfs embed.FS

var last_reached time.Time

type ReachResult struct {
	Destination string
	Reached     bool
}

var reach_results []ReachResult

func main() {
	last_reached = time.Unix(0, 0)
	// Parse config.toml
	cfg, err := Unmarshal()

	if err != nil {
		panic(err)
	}

	engine := html.NewFileSystem(http.FS(viewsfs), ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		// Retrieve server information
		/// General informations
		v_mem, _ := mem.VirtualMemory()
		l, _ := load.Avg()
		h, _ := host.BootTime()
		btFromUnix := time.Unix(int64(h), 0)
		/// Machine temperature
		t, _ := host.SensorsTemperatures()
		var temperature string
		if len(t) > 0 {
			temperature = fmt.Sprintf("%.2f ℃", t[0].Temperature)
		} else {
			temperature = "Not available."
		}

		var ip_version string
		if isIpv6(c.IP()) {
			ip_version = "IPv6"
		} else {
			ip_version = "IPv4"
		}

		return c.Render("views/index", fiber.Map{
			"location_id":     cfg.General.LocationId,
			"location_pretty": cfg.General.LocationPretty,
			"domain":          cfg.General.Domain,
			"loadavg":         fmt.Sprintf("%.2f, %.2f, %.2f", l.Load1, l.Load5, l.Load15),
			"uptime":          btFromUnix,
			"ram_used":        fmt.Sprintf("%.2f", v_mem.UsedPercent),
			"client_source":   ip_version,
			"temperature":     temperature,
		})
	})

	app.Get("/reach", func(c *fiber.Ctx) error {
		if time.Now().Sub(last_reached) > 5*time.Minute {
			dest := cfg.Http.Destinations

			for i := 0; i < len(dest); i++ {
				is200 := false
				resp, err := http.Get(dest[i])

				if err == nil {
					if resp.StatusCode == 200 {
						is200 = true
					}
				}

				result := ReachResult{
					Destination: dest[i],
					Reached:     is200,
				}

				reach_results = append(reach_results, result)
				last_reached = time.Now()
			}

		}
		u, err := json.Marshal(reach_results)
		if err != nil {
			panic(err)
		}

		return c.SendString(string(u))
	})

	log.Fatal(app.Listen(":3000"))
}
