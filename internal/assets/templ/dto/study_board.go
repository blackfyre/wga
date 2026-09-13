package dto

// StudyBoardWork is the public, published artwork projection shared by the
// comparison page and the transient fixed shelf.
type StudyBoardWork struct {
	ID         string
	Title      string
	Artist     string
	URL        string
	ImageURL   string
	Date       string
	Dimensions string
	Medium     string
	Location   string
	School     string
	Form       string
	Type       string
}
