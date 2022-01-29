package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
	_ "time/tzdata"

	"github.com/ingmarstein/velux-nibe/nibe"
	"github.com/ingmarstein/velux-nibe/velux"
)

func lenientParseInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

var username = flag.String("velux-user", os.Getenv("VELUX_USERNAME"), "Velux user name")
var password = flag.String("velux-password", os.Getenv("VELUX_PASSWORD"), "Velux password")
var clientID = flag.String("nibe-client-id", os.Getenv("NIBE_CLIENT_ID"), "NIBE Uplink client ID")
var clientSecret = flag.String("nibe-client-secret", os.Getenv("NIBE_CLIENT_SECRET"), "NIBE Uplink client secret")
var callbackURL = flag.String("nibe-callback", os.Getenv("NIBE_CALLBACK_URL"), "NIBE Uplink callback URL")
var system = flag.Int("nibe-system", lenientParseInt(os.Getenv("NIBE_SYSTEM_ID")), "Nibe system id")
var nibeTokenFile = flag.String("nibe-token", os.Getenv("NIBE_TOKEN"), "File name to store the Nibe token")
var verbose = flag.Bool("verbose", false, "Verbose mode")
var targetTemp = flag.Int("targetTemp", 210, "Target temperature in celsius, multiplied by ten")
var pollInterval = flag.Int("interval", 60, "Polling interval in seconds")

// https://medium.com/@mhcbinder/using-local-time-in-a-golang-docker-container-built-from-scratch-2900af02fbaf
func updateTimezone() {
	if tz := os.Getenv("TZ"); tz != "" {
		var err error
		time.Local, err = time.LoadLocation(tz)
		if err != nil {
			log.Printf("error loading location '%s': %v\n", tz, err)
		}
	}
}

func main() {
	updateTimezone()

	flag.Parse()

	if *username == "" ||
		*password == "" ||
		*clientID == "" ||
		*clientSecret == "" ||
		*callbackURL == "" ||
		*system == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *nibeTokenFile == "" {
		*nibeTokenFile = "nibe-token.json"
	}

	log.Println("Creating NIBE client")
	nibeClient := nibe.NewClientWithAuth(*clientID, *clientSecret, *callbackURL, *nibeTokenFile, []string{nibe.ScopeWrite})
	nibeClient.Verbose = *verbose

	log.Println("Creating Velux client")
	veluxClient := velux.NewClientWithAuth(*username, *password)
	veluxClient.Verbose = *verbose

	ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)

	for {
		select {
		case <-ticker.C:
			homeData, err := veluxClient.GetHomesData(velux.GetHomesDataRequest{GatewayTypes: []string{velux.Bridge}})
			if err != nil {
				log.Printf("error getting home data: %v", err)
				continue
			}

			for _, home := range homeData.Body.Homes {
				roomNames := make(map[string]string)
				for _, room := range home.Rooms {
					roomNames[room.ID] = room.Name
				}

				status, err := veluxClient.HomeStatus(velux.HomeStatusRequest{
					HomeID:      home.ID,
					DeviceTypes: []string{velux.Sensor},
				})
				if err != nil {
					log.Printf("error getting home status: %v", err)
					continue
				}
				for _, room := range status.Body.Home.Rooms {
					roomName, ok := roomNames[room.ID]
					if !ok {
						roomName = room.ID
					}

					log.Printf("Home %s - room %s - temperature %d", home.Name, roomName, room.Temperature)
					externalId, err := strconv.Atoi(room.ID)
					if err != nil {
						log.Printf("Home %s - room %s: failed to parse room ID: %v", home.Name, roomName, err)
						continue
					}
					nibeClient.SetThermostat(nibe.SetThermostatRequest{
						SystemID:       *system,
						ExternalId:     externalId,
						Name:           roomName,
						ActualTemp:     room.Temperature,
						TargetTemp:     *targetTemp,
						ClimateSystems: []int{1},
					})
				}
			}
		}
	}
}
