package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

type Song struct {
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

func ParseMusicListToUrls(filePath string) ([]string, error) {
	fmt.Println("Parsing music list to urls...")

	var data []Century

	// Read the data from the file
	fileData, err := os.ReadFile(filePath)

	if err != nil {
		fmt.Println("Error reading file:", err)
	}

	// Unmarshal the JSON data into the data variable
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data:", err)
	}

	fmt.Println("Done reading file")

	// TODO: add id to each song

	var parsedData []string
	for _, century := range data {
		for _, composer := range century.Composers {
			for _, song := range composer.Songs {
				if len(song.Source) > 0 {
					for _, source := range song.Source {
						url := fmt.Sprintf("%s", source)
						parsedData = append(parsedData, url)
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
	}
	defer file.Close()

	jsonData, err := json.Marshal(parsedData)
	if err != nil {
		fmt.Println("Error marshalling JSON data:", err)
	}

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing JSON data to file:", err)
	}

	return parsedData, nil
}
