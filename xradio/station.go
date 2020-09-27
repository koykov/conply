package main

import "strings"

type Station struct {
	Alias   string
	Key     string
	Station string
}

type Stations []Station

func (s *Stations) Look(alias string) *Station {
	for _, st := range *s {
		if st.Alias == alias {
			return &st
		}
	}
	return nil
}

func (s *Stations) PrettyPrint() string {
	list := make([]string, 0)
	for _, st := range *s {
		list = append(list, "  "+st.Alias)
	}
	return strings.Join(list, "\n")
}
