package startup

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	reset  = "\033[0m"
	dim    = "\033[2m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
)

var logo = []string{
	"",
	"  ██       ██  ████   ████   ██  ██ ██",
	"  ██       ██ ██  ██  ██ ██  ██  ██ ██",
	"  ██  ██ ██ ██████  ██ ██  ██  ██ ██",
	"  ██ ██ ██ ██ ██  ██ ██  ████  ██ ██",
	"   ███ ███  ██  ███  ████   ██   ████",
	"",
}

// PrintBanner prints the LumeHub startup banner and runtime info.
func PrintBanner(addr, dataDir, wwwDir string, authEnabled bool) {
	colored := useColor()

	fmt.Println()
	for _, line := range logo {
		fmt.Println(paint(colored, cyan, line))
	}
	fmt.Println(paint(colored, cyan, "              光盒 · LumeHub"))
	fmt.Println(paint(colored, dim, "            收纳影像，寄存私藏"))
	fmt.Println()

	fmt.Println(paint(colored, green, "  ● Server ready"))
	localURL, networkURL := primaryURLs(addr)
	fmt.Printf("    %s Local    %s\n", paint(colored, dim, "›"), paint(colored, yellow, localURL))
	if networkURL != "" && networkURL != localURL {
		fmt.Printf("    %s Network  %s\n", paint(colored, dim, "›"), paint(colored, yellow, networkURL))
	}
	// fmt.Printf("    %s Data     %s\n", paint(colored, dim, "›"), paint(colored, dim, dataDir))
	// fmt.Printf("    %s WWW      %s\n", paint(colored, dim, "›"), paint(colored, dim, wwwDir))
	// if authEnabled {
	// 	fmt.Printf("    %s Auth     %s\n", paint(colored, dim, "›"), paint(colored, green, "enabled"))
	// } else {
	// 	fmt.Printf("    %s Auth     %s\n", paint(colored, dim, "›"), paint(colored, dim, "disabled (dev mode)"))
	// }
	fmt.Println()
}

func useColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("LUMEHUB_NO_COLOR"), "1") || strings.EqualFold(os.Getenv("LUMEHUB_NO_COLOR"), "true") {
		return false
	}
	return true
}

func paint(enabled bool, code, text string) string {
	if !enabled {
		return text
	}
	return code + text + reset
}

func primaryURLs(addr string) (localURL, networkURL string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr, ""
	}

	host = strings.Trim(host, "[]")
	switch {
	case host == "" || host == "0.0.0.0" || host == "::":
		localURL = fmt.Sprintf("http://127.0.0.1:%s", port)
		if ip := primaryLocalIPv4(); ip != "" {
			networkURL = fmt.Sprintf("http://%s:%s", ip, port)
		}
	case host == "localhost" || host == "127.0.0.1" || host == "::1":
		localURL = fmt.Sprintf("http://127.0.0.1:%s", port)
	case strings.Contains(host, ":"):
		localURL = fmt.Sprintf("http://[%s]:%s", host, port)
	default:
		localURL = fmt.Sprintf("http://%s:%s", host, port)
	}
	return localURL, networkURL
}

func primaryLocalIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || isVirtualInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
				continue
			}
			return ipNet.IP.String()
		}
	}
	return ""
}

func isVirtualInterface(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{
		"vethernet", "vmware", "virtualbox", "virtual", "hyper-v",
		"wsl", "tap", "tun", "npcap", "bluetooth", "loopback",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
