package utils

import (
	"fmt"
	"log"
	"strings"

	"blackfyre.ninja/wga/models"
	"github.com/pocketbase/pocketbase"
)

func ApplyGlossary(app *pocketbase.PocketBase, content string) string {
	gi, err := models.GetGlossaryItems(app.Dao())
	if err != nil {
		log.Printf("Error getting glossary items: %s", err)
	}

	toAppend := ""

	for _, item := range gi {
		item.Expression = strings.ToLower(item.Expression) // make sure we're comparing apples to apples

		if strings.Contains(content, item.Expression) {
			// content has html in it, we only need to replace item.Expression where the match is not part of an html tag attribute

			instances := strings.Count(content, item.Expression)

			indexTracker := 0

			// content has more than one instance of item.Expression, we need to replace each instance
			for i := 0; i < instances; i++ {

				// First, we need to find the index of the first occurrence of item.Expression in content
				// We'll use this index to determine if the match is part of an html tag attribute
				index := strings.Index(content[indexTracker:], item.Expression)

				// Next, we need to find the index of the last occurrence of "<" before the first occurrence of item.Expression
				// We'll use this index to determine if the match is part of an html tag attribute
				lastIndex := strings.LastIndex(content[indexTracker:indexTracker+index], "<")

				// If the last occurrence of "<" is greater than the last occurrence of ">", then the match is part of an html tag attribute
				// In this case, we don't want to replace the match
				if strings.LastIndex(content[indexTracker:indexTracker+index], ">") < lastIndex {
					indexTracker = indexTracker + index + 1
					continue
				}

				// If the match is not part of an html tag attribute, we can replace it
				// We'll use the index of the first occurrence of item.Expression in content to replace the match
				content = strings.Replace(content, item.Expression, fmt.Sprintf("<span data-tt-content=\"%s\">%s</span>", item.Id, item.Expression), 1)

				indexTracker = indexTracker + index + 1
			}

			// Append the template with item.Id and item.Definition
			toAppend = toAppend + fmt.Sprintf("<template  id=\"%s\"><div class=\"content\">%s</div></template>", item.Id, item.Definition)
		}
	}

	return content + toAppend
}
