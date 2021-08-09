package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

var hostnameCache string
var ipAddressCache string
var targetStringCache string

func init() {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	hostnameCache = hostname

	// Get IP address
	conn, error := net.Dial("udp", "8.8.8.8:80")  
	if error != nil {  
		fmt.Println(error)
		os.Exit(1)
	}  
	defer conn.Close()  
	ipAddress := conn.LocalAddr().(*net.UDPAddr).IP 
	ipAddressCache = ipAddress.String()

	targetStringCache = os.Getenv("TARGET")

	fmt.Printf("[%s] Starting, hostname: %s, IP: %s, port: 8080, %s\n", 
		getCurrentTime(), hostnameCache, ipAddressCache, targetStringCache)

}

func main() {
	http.HandleFunc("/", mainHandler)
	http.ListenAndServe(":8080", nil)
}

func getCurrentTime() string {
	dt := time.Now()
	return dt.Format("01-01-2006 15:04:05.000")
}

func mainHandler(res http.ResponseWriter, req *http.Request) {
	outputStr := fmt.Sprintf("[%s] Hello from hostname: %s, IP: %s, %s\n", 
		getCurrentTime(), hostnameCache, ipAddressCache, targetStringCache)
	fmt.Printf(outputStr)
	data := []byte(outputStr)
	res.WriteHeader(200)
	res.Write(data)
}