package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/BurntSushi/toml"
	sigar "github.com/cloudfoundry/gosigar"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
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

func main() {
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
		uptime := sigar.Uptime{}
		uptime.Get()
		loadavg := sigar.LoadAverage{}
		loadavg.Get()
		mem := sigar.Mem{}
		mem.Get()

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
			"loadavg":         fmt.Sprintf("%.2f, %.2f, %.2f", loadavg.One, loadavg.Five, loadavg.Fifteen),
			"uptime":          uptime.Format(),
			"free_ram":        ByteCountDecimal(int64(mem.Free)),
			"client_source":   ip_version,
		})
	})

	app.Get("/reach", func(c *fiber.Ctx) error {
		type Result struct {
			Destination string
			Reached     bool
		}

		dest := cfg.Http.Destinations
		results := []Result{}

		for i := 0; i < len(dest); i++ {
			is200 := false
			resp, err := http.Get(dest[i])

			if err == nil {
				if resp.StatusCode == 200 {
					is200 = true
				}
			}

			result := Result{
				Destination: dest[i],
				Reached:     is200,
			}

			results = append(results, result)
		}

		u, err := json.Marshal(results)
		if err != nil {
			panic(err)
		}

		return c.SendString(string(u))
	})

	log.Fatal(app.Listen(":3000"))
}
