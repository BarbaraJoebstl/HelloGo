package main

import (
      "encoding/json"
      "net/http"
      "strings"
      "time"
      "log"
)

func main() {
    http.HandleFunc("/hello", hello)

    mw := multiWeatherProvider{
       openWeatherMap{apiKey: "363a7474d209434e6c81c1b56e232b21" },
       weatherUnderground{apiKey: "your-key-here"},
   }

   http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
       begin := time.Now()
       city := strings.SplitN(r.URL.Path, "/", 3)[2]

       temp, err := mw.temperature(city)
       if err != nil {
           http.Error(w, err.Error(), http.StatusInternalServerError)
           return
       }

       w.Header().Set("Content-Type", "application/json; charset=utf-8")
       json.NewEncoder(w).Encode(map[string]interface{}{
           "city": city,
           "temp": temp,
           "took": time.Since(begin).String(),
       })
   })

   http.ListenAndServe(":8080", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello!"))
}

// for the open weather map api http://api.openweathermap.org/
type openWeatherMap struct{
  apiKey string
}

// for the open weather map api //www.wunderground.com
type weatherUnderground struct {
    apiKey string
}

type weatherData struct {
    Name string `json:"name"`
    Main struct {
        Kelvin float64 `json:"temp"`
    } `json:"main"`
}

// same behaviour for all weather apis
type weatherProvider interface {
    temperature(city string) (float64, error) // in Kelvin, naturally
}

func (w openWeatherMap) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=" + w.apiKey + "&q=" + city)
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
    return d.Main.Kelvin, nil
}

func (w weatherUnderground) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json")
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
        Observation struct {
            Celsius float64 `json:"temp_c"`
        } `json:"current_observation"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    kelvin := d.Observation.Celsius + 273.15
    log.Printf("weatherUnderground: %s: %.2f", city, kelvin)
    return kelvin, nil
}

func temperature(city string, providers ...weatherProvider) (float64, error) {
    sum := 0.0

    for _, provider := range providers {
        k, err := provider.temperature(city)
        if err != nil {
            return 0, err
        }

        sum += k
    }

    return sum / float64(len(providers)), nil
}

type multiWeatherProvider []weatherProvider

func (w multiWeatherProvider) temperature(city string) (float64, error) {
    sum := 0.0

    for _, provider := range w {
        k, err := provider.temperature(city)
        if err != nil {
            return 0, err
        }

        sum += k
    }

    return sum / float64(len(w)), nil
}
