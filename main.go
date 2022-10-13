package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
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
var system = flag.Int("nibe-system", lenientParseInt(os.Getenv("NIBE_SYSTEM_ID")), "NIBE system ID")
var nibeTokenFile = flag.String("nibe-token", os.Getenv("NIBE_TOKEN"), "File name to store the NIBE token")
var verbose = flag.Bool("verbose", false, "Verbose mode")
var targetTemp = flag.Int("targetTemp", 210, "Target temperature in celsius, multiplied by ten")
var pollInterval = flag.Int("interval", 60, "Polling interval in seconds")
var httpPort = flag.Int("http-port", lenientParseInt(os.Getenv("HTTP_PORT")), "Port for HTTP interface (0 = disabled)")
var configFile = flag.String("conf", "", "Config file")

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

type UpdateResult struct {
	Timestamp         time.Time
	Name              string
	ActualTemperature int
	TargetTemperature int
	Result            error
}

type SystemSettings struct {
	Username          string `json:"velux_user"`
	Password          string `json:"velux_password"`
	ClientID          string `json:"nibe_client_id"`
	ClientSecret      string `json:"nibe_client_secret"`
	CallbackURL       string `json:"nibe_callback"`
	System            int    `json:"nibe_system"`
	TokenFile         string `json:"nibe_token"`
	PollInterval      int    `json:"interval"`
	Verbose           bool   `json:"verbose"`
	TargetTemperature int    `json:"target_temperature"`
	HTTPPort          int    `json:"http_port,omitempty"`
}

type SystemState struct {
	SettingsMu sync.RWMutex
	Settings   SystemSettings

	UpdatesMu  sync.RWMutex
	LastUpdate []UpdateResult
}

var htmlTemplate = template.Must(template.New("main").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Velux-Nibe</title>
	</head>
	<body>
		<h1>Velux-Nibe</h1>
		<h2>Configuration</h2>
		<table>
			<tr><td>Velux user</td><td>{{.Settings.Username}}</td></tr>
			<tr><td>NIBE client ID</td><td>{{.Settings.ClientID}}</td></tr>
			<tr><td>NIBE system</td><td>{{.Settings.System}}</td></tr>
			<tr><td>Poll interval</td><td>{{.Settings.PollInterval}}</td></tr>
		</table>
		<h2>Settings</h2>
		<form method="POST" action="/">
			<label for="target_temperature">Target temperature:</label>
			<input type="text" name="target_temperature" value="{{.Settings.TargetTemperature}}">
			<input type="submit" value="submit" />
		</form>
		<h2>Last Update</h2>
		{{range .LastUpdate}}
		<h3>Room {{.Name}}</h3>
		<table>
			<tr><td>Timestamp</td><td>{{.Timestamp.Format "Jan 02, 2006 15:04:05 UTC"}}</td></tr>
			<tr><td>Actual temperature</td><td>{{.ActualTemperature}}</td></tr>
			<tr><td>Target temperature</td><td>{{.TargetTemperature}}</td></tr>
			<tr><td>Result</td><td>{{if .Result}}{{.Result}}{{else}}success{{end}}</td></tr>
		</table>
		{{end}}
  </body>
</html>
`))

func (state *SystemState) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		newTempString := r.FormValue("target_temperature")
		newTemp, err := strconv.Atoi(newTempString)
		if err != nil {
			fmt.Fprintf(w, "Invalid temperature: %v", err)
			return
		}

		if newTemp < 100 || newTemp > 300 {
			fmt.Fprintf(w, "Invalid temperature: %d (must be between 100 (10.0 °C) and 300 (30.0 °C)", newTemp)
			return
		}

		state.SettingsMu.Lock()
		defer state.SettingsMu.Unlock()
		state.Settings.TargetTemperature = newTemp

		if *configFile != "" {
			f, err := os.OpenFile(*configFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				log.Printf("Unable to write config file: %v", err)
			} else {
				enc := json.NewEncoder(f)
				enc.SetIndent("", "  ")
				enc.Encode(state.Settings)
				f.Close()
			}
		}
	}

	state.UpdatesMu.RLock()
	defer state.UpdatesMu.RUnlock()

	if err := htmlTemplate.Execute(w, state); err != nil {
		log.Fatal(err)
	}
}

func main() {
	updateTimezone()

	flag.Parse()

	var flagsPassed = make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsPassed[f.Name] = true
	})

	state := SystemState{Settings: SystemSettings{}}

	if *configFile != "" {
		f, err := os.Open(*configFile)
		if err != nil {
			log.Fatalf("Failed to open config file: %v", err)
		}
		err = json.NewDecoder(f).Decode(&state.Settings)
		if err != nil {
			log.Fatalf("Failed to parse config file: %v", err)
		}
		f.Close()
	}

	// command line flags override settings from the config file
	if *username != "" {
		state.Settings.Username = *username
	}
	if *password != "" {
		state.Settings.Password = *password
	}
	if *clientID != "" {
		state.Settings.ClientID = *clientID
	}
	if *clientSecret != "" {
		state.Settings.ClientSecret = *clientSecret
	}
	if *callbackURL != "" {
		state.Settings.CallbackURL = *callbackURL
	}
	if *nibeTokenFile != "" {
		state.Settings.TokenFile = *nibeTokenFile
	}
	if *system != 0 {
		state.Settings.System = *system
	}
	if flagsPassed["interval"] {
		state.Settings.PollInterval = *pollInterval
	}
	if *verbose {
		state.Settings.Verbose = true
	}
	if flagsPassed["targetTemp"] {
		state.Settings.TargetTemperature = *targetTemp
	}
	if *httpPort != 0 {
		state.Settings.HTTPPort = *httpPort
	}

	if state.Settings.Username == "" ||
		state.Settings.Password == "" ||
		state.Settings.ClientID == "" ||
		state.Settings.ClientSecret == "" ||
		state.Settings.CallbackURL == "" ||
		state.Settings.System == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if state.Settings.TokenFile == "" {
		state.Settings.TokenFile = "nibe-token.json"
	}

	log.Println("Creating NIBE client")
	nibeClient := nibe.NewClientWithAuth(state.Settings.ClientID, state.Settings.ClientSecret, state.Settings.CallbackURL, state.Settings.TokenFile, []string{nibe.ScopeWrite})
	nibeClient.Verbose = state.Settings.Verbose

	log.Println("Creating Velux client")
	veluxClient := velux.NewClientWithAuth(state.Settings.Username, state.Settings.Password)
	veluxClient.Verbose = state.Settings.Verbose

	if state.Settings.HTTPPort != 0 {
		http.HandleFunc("/", state.Handler)
		go func() {
			if err := http.ListenAndServe(fmt.Sprintf(":%d", state.Settings.HTTPPort), nil); err != nil {
				log.Fatal(err)
			}
		}()
	}

	ticker := time.NewTicker(time.Duration(state.Settings.PollInterval) * time.Second)

	for ; true; <-ticker.C {
		homeData, err := veluxClient.GetHomesData(velux.GetHomesDataRequest{GatewayTypes: []string{velux.Bridge}})
		if err != nil {
			log.Printf("error getting home data: %v", err)
			continue
		}

		var updates []UpdateResult

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
				state.SettingsMu.RLock()
				temp := state.Settings.TargetTemperature
				state.SettingsMu.RUnlock()
				err = nibeClient.SetThermostat(nibe.SetThermostatRequest{
					SystemID:       state.Settings.System,
					ExternalId:     externalId % math.MaxInt32, // The NIBE Uplink API doesn't accept values > 2^31
					Name:           roomName,
					ActualTemp:     room.Temperature,
					TargetTemp:     temp,
					ClimateSystems: []int{1},
				})
				updates = append(updates, UpdateResult{
					Timestamp:         time.Now(),
					Name:              roomName,
					ActualTemperature: room.Temperature,
					TargetTemperature: temp,
					Result:            err,
				})
				if err != nil {
					log.Printf("Failed to set thermostat %d in room %s: %v", externalId, roomName, err)
				}
			}
			state.UpdatesMu.Lock()
			state.LastUpdate = updates
			state.UpdatesMu.Unlock()
		}
	}
}
