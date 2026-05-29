package oui

import (
	"encoding/csv"
	_ "embed"
	"fmt"
	"net"
	"strings"
)

//go:embed oui.csv
var ouiCSV string

var ouiMap map[string]string

func init() {
	ouiMap = make(map[string]string)
	r := csv.NewReader(strings.NewReader(ouiCSV))
	records, err := r.ReadAll()
	if err != nil {
		panic(fmt.Sprintf("oui: failed to parse embedded CSV: %v", err))
	}
	for _, rec := range records[1:] { // skip header
		if len(rec) < 3 {
			continue
		}
		assignment := strings.ToUpper(strings.TrimSpace(rec[1]))
		orgName := strings.TrimSpace(rec[2])
		ouiMap[assignment] = orgName
	}
}

// Lookup returns the organization name for the OUI prefix of mac, or "Unknown".
func Lookup(mac net.HardwareAddr) string {
	if len(mac) < 3 {
		return "Unknown"
	}
	key := fmt.Sprintf("%02X%02X%02X", mac[0], mac[1], mac[2])
	if name, ok := ouiMap[key]; ok {
		return name
	}
	return "Unknown"
}

// Count returns the number of OUI entries loaded.
func Count() int {
	return len(ouiMap)
}
