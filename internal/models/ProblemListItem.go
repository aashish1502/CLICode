package models

type ProblemListItem struct {
	Id               int      `json:"id"`
	Title            string   `json:"title"`
	Difficulty       string   `json:"difficulty"`
	Tags             []string `json:"tags"`
	Platform         string   `json:"platform"`
	Url              string   `json:"url"`
	Solved           bool     `json:"solved"`
	Attempts         int      `json:"attempts"`
	TimeTakenMinutes int      `json:"time_taken_minutes"`
	DateSolved       string   `json:"date_solved"`
	Review           bool     `json:"review"`
	Notes            string   `json:"notes"`
}
