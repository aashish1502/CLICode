package models

import "time"

type ProblemListItem struct {
	Id                 string    `json:"id"`
	Title              string    `json:"title"`
	Difficulty         string    `json:"difficulty"`
	Tags               []string  `json:"tags"`
	Platform           string    `json:"platform"`
	Url                string    `json:"url"`
	Solved             bool      `json:"solved"`
	Attempts           int       `json:"attempts"`
	TimeTakenInMinutes int       `json:"time_taken_in_minutes"`
	DateSolved         time.Time `json:"date_solved"`
	Review             bool      `json:"review"`
	Notes              []string  `json:"notes"`
}
