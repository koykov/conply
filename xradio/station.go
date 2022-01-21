package main

import "strings"

type Station struct {
	Alias   string
	Key     string
	Station string
	API     string
}

type Stations []Station

// Search station alias in registry of stations.
func (s *Stations) Look(alias string) *Station {
	for _, st := range *s {
		if st.Alias == alias {
			return &st
		}
	}
	return nil
}

// Build a human readable list of a stations.
func (s *Stations) PrettyPrint() string {
	list := make([]string, 0)
	for _, st := range *s {
		list = append(list, "  "+st.Alias)
	}
	return strings.Join(list, "\n")
}

// Implement fmt.Stringer
func (s *Station) String() string {
	return s.Station
}
