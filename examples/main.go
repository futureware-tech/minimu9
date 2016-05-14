package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dasfoo/i2c"
	"github.com/dasfoo/minimu9"
	"github.com/dasfoo/minimu9/l3gd"
	"github.com/dasfoo/minimu9/lsm303d"
	"github.com/golang/geo/r3"

	"golang.org/x/net/websocket"
)

var bus i2c.Bus

func socketHandler(ws *websocket.Conn) {
	a := lsm303d.NewAccelerometer(bus, lsm303d.DefaultAddress)
	if e := a.Wake(); e != nil {
		log.Fatal(e)
	}
	m := lsm303d.NewMagnetometer(bus, lsm303d.DefaultAddress)
	if e := m.Wake(); e != nil {
		log.Fatal(e)
	}
	g := l3gd.NewGyro(bus, l3gd.DefaultAddress)
	if e := g.Wake(); e != nil {
		log.Fatal(e)
	}

	var v struct {
		A, M, G, H, MR r3.Vector
		E              string
	}

	dataDelay := 33

	go func() {
		data := make([]byte, 255)
		var e error
		for {
			if size, err := ws.Read(data); err != nil {
				fmt.Println("Read:", err)
				break
			} else {
				nameAndValue := strings.Split(string(data[:size]), "=")
				switch nameAndValue[0] {
				case "frequency":
					v, _ := strconv.Atoi(nameAndValue[1])
					dataDelay = 1000 / v

				case "M.frequency":
					v, _ := strconv.ParseFloat(nameAndValue[1], 32)
					e = m.SetFrequency(byte(v))
				case "M.full_scale":
					v, _ := strconv.Atoi(nameAndValue[1])
					e = m.SetFullScale(float64(v))
				case "M.power_down":
					v, _ := strconv.ParseBool(nameAndValue[1])
					if v {
						e = m.Sleep()
					} else {
						e = m.Wake()
					}
				case "M.calibrate":
					stop := make(chan int)
					go func() {
						var e error
						v.MR, e = m.Calibrate(stop)
						if e != nil {
							v.E = e.Error()
						}
					}()
					time.Sleep(30 * time.Second)
					stop <- 0

				case "A.frequency":
					v, _ := strconv.ParseFloat(nameAndValue[1], 32)
					e = a.SetFrequency(v)
				case "A.full_scale":
					v, _ := strconv.Atoi(nameAndValue[1])
					e = a.SetFullScale(float64(v))
				case "A.antialias_bandwidth":
					v, _ := strconv.Atoi(nameAndValue[1])
					e = a.SetAntiAliasBandwidth(uint16(v))
				case "A.power_down":
					v, _ := strconv.ParseBool(nameAndValue[1])
					if v {
						e = a.Sleep()
					} else {
						e = a.Wake()
					}

				case "G.frequency":
					v, _ := strconv.ParseFloat(nameAndValue[1], 32)
					e = g.SetFrequency(v)
				case "G.full_scale":
					v, _ := strconv.Atoi(nameAndValue[1])
					e = g.SetFullScale(float64(v))
				case "G.power_down":
					v, _ := strconv.ParseBool(nameAndValue[1])
					if v {
						e = g.Sleep()
					} else {
						e = g.Wake()
					}
				case "G.calibrate":
					stop := make(chan int)
					go g.Calibrate(stop)
					time.Sleep(5 * time.Second)
					stop <- 0
				default:
					v.E = "Unknown parameter: " + nameAndValue[0]
				}
				if e != nil {
					v.E = "Setting " + nameAndValue[0] + ": " + e.Error()
					e = nil
				}
			}
		}
	}()

	var e error
	for {
		if v.A, e = a.Read(); e != nil {
			if _, ok := e.(*minimu9.DataAvailabilityError); !ok {
				log.Fatal(e)
			}
		}
		if v.M, e = m.Read(); e != nil {
			if _, ok := e.(*minimu9.DataAvailabilityError); !ok {
				log.Fatal(e)
			}
		}
		if v.G, e = g.Read(); e != nil {
			if _, ok := e.(*minimu9.DataAvailabilityError); !ok {
				log.Fatal(e)
			}
		}

		v.H = r3.Vector{
			X: 0,
			Y: math.Atan2(v.M.X, v.M.Y),
			Z: 0,
		}

		var data []byte
		if data, e = json.Marshal(v); e != nil {
			log.Fatal(e)
		}
		if _, e := ws.Write(data); e != nil {
			fmt.Println(e)
			return
		}
		time.Sleep(time.Duration(dataDelay) * time.Millisecond)
	}
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func main() {
	var e error

	if bus, e = i2c.NewBus(1); e != nil {
		log.Fatal(e)
	}

	port := ":8080"
	http.Handle("/minimu9", websocket.Handler(socketHandler))
	http.Handle("/", http.FileServer(http.Dir("/tmp/minimu9")))
	if ip, err := externalIP(); err == nil {
		fmt.Printf("Starting server at http://%s%s\n", ip, port)
	}
	err := http.ListenAndServe(port, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
