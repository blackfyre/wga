package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

type UrlData struct {
    Url string
    ID  uuid.UUID
}

type Song struct {
	ID     uuid.UUID
	Title  string
	URL    string
	Source []string
}

type Composer struct {
	Name     string
	Date     string
	Language string
	Songs    []Song
}

type Century struct {
	Century   string
	Composers []Composer
}

func ParseMusicListToUrls(filePath string) ([]UrlData, error) {
	fmt.Println("Parsing music list to urls...")

	var data []Century

	// Read the data from the file
	fileData, err := os.ReadFile(filePath)

	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}

	// Unmarshal the JSON data into the data variable
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data:", err)
		return nil, err
	}

	var parsedData []UrlData
	for _, century := range data {
		for _, composer := range century.Composers {
			for _, song := range composer.Songs {
				if len(song.Source) > 0 {
					for _, source := range song.Source {
						urlData := UrlData{
							Url: source,
							ID:  uuid.New(),
						}
						parsedData = append(parsedData, urlData)
					}
				}
			}
		}
	}

	fmt.Println("Done parsing music list to urls.")

	// Write the parsed data to a JSON file
	file, err := os.Create("musicUrls.json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return nil, err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Println("Error closing file:", cerr)
		}
	}()

	jsonData, err := json.Marshal(parsedData)
	if err != nil {
		fmt.Println("Error marshalling JSON data:", err)
		return nil, err
	}

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing JSON data to file:", err)
		return nil, err
	}

	return parsedData, nil
}
