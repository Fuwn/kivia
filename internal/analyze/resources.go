package analyze

import (
	"fmt"
	"github.com/Fuwn/kivia/internal/nlp"
)

type resources struct {
	dictionary *nlp.Dictionary
}

func getResources() (resources, error) {
	return loadResources()
}

func loadResources() (resources, error) {
	dictionary, err := nlp.NewDictionary()

	if err != nil {
		return resources{}, fmt.Errorf("Failed to load dictionary: %w", err)
	}

	return resources{
		dictionary: dictionary,
	}, nil
}
