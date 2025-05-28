package dto

type GuestbookEntry struct {
	Name     string
	Message  string
	Email    string
	Location string
	Created  string
}

type GuestbookEntries []GuestbookEntry
