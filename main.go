package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type (
	GeneralSection struct {
		Location string `toml:"location"`
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
	content, err := os.ReadFile("./config.toml")
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

	hostname := ""
	if Exists("/etc/host_hostname") {
		bytes, err := os.ReadFile("/etc/host_hostname")
		if err != nil {
			return
		}
		hostname = string(bytes)
	} else {
		hostname, _ = os.Hostname()
	}

	cpus, err := cpu.Info()

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

		var ip_version string
		if isIpv6(c.IP()) {
			ip_version = "IPv6"
		} else {
			ip_version = "IPv4"
		}

		l, _ := load.Avg()

		client_ip := c.IP()
		headers := c.GetReqHeaders()
		forwarded, ok := headers["X-Forwarded-For"]
		if ok {
			client_ip = forwarded
		}

		remote_addr, err := net.LookupAddr(client_ip)
		if err != nil {
			remote_addr[0] = "N/A"
		}
		if len(remote_addr) == 0 {
			remote_addr[0] = "N/A"
		}

		return c.Render("views/index", fiber.Map{
			"cpu":           cpus[0].ModelName,
			"platform":      runtime.GOOS,
			"arch":          runtime.GOARCH,
			"location":      cfg.General.Location,
			"hostname":      strings.Replace(hostname, "s-", "", -1),
			"loadavg":       fmt.Sprintf("%.2f, %.2f, %.2f", l.Load1, l.Load5, l.Load15),
			"ram_used":      fmt.Sprintf("%.2f", v_mem.UsedPercent),
			"client_source": ip_version,
			"client_ip":     client_ip,
			"client_port":   c.Port(),
			"client_host":   remote_addr[0],
		})
	})

	app.Get("/reach", func(c *fiber.Ctx) error {
		if time.Now().Sub(last_reached) > 5*time.Minute {
			reach_results = nil
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

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
