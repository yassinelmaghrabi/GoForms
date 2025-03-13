package models

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func readJson(path string) QuestionsData {
	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	var questionsData QuestionsData
	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &questionsData)
	if err != nil {
		fmt.Println(err)
	}

	return questionsData
}

type MCQChoices map[string]string

type Question struct {
	ID       int         `json:"id"`
	Question string      `json:"question"`
	Type     string      `json:"type"`
	Unit     string      `json:"unit,omitempty"`
	Page     string      `json:"page,omitempty"`
	Language string      `json:"language,omitempty"`
	Options  interface{} `json:"options,omitempty"`
}

type HtmlQuestion struct {
	Question Question
	HTML     string
}

type QuestionsData struct {
	Questions []Question `json:"questions"`
}

type Option struct {
	Key   string
	Value interface{}
}

func generateHTML(q Question) string {
	switch q.Type {
	case "categorical":
		options, ok := q.Options.([]interface{})
		if !ok {
			return ""
		}

		var sb strings.Builder
		sb.WriteString(`<select class="block text-white appearance-auto bg-gray-700 rounded-sm my-4 p-2" name="question_` + fmt.Sprint(q.ID) + `">`)
		for _, opt := range options {
			sb.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, opt, opt))
		}
		sb.WriteString("</select>")
		return sb.String()

	case "MCQ":
		options, ok := q.Options.(map[string]interface{})
		if !ok {
			return ""
		}

		var optionSlice []Option
		for key, value := range options {
			optionSlice = append(optionSlice, Option{Key: key, Value: value})
		}

		sort.Slice(optionSlice, func(i, j int) bool {
			return optionSlice[i].Key < optionSlice[j].Key
		})

		var sb strings.Builder
		sb.WriteString("<ul>")
		for _, opt := range optionSlice {
			sb.WriteString(fmt.Sprintf(`<li class="block text-white my-3"><input class="accent-pink-500" type="radio" name="question_%d" value="%s"> %s</li>`, q.ID, opt.Key, opt.Value))
		}
		sb.WriteString("</ul>")
		return sb.String()

	case "number":
		return fmt.Sprintf(`<input class="block text-white appearance-auto bg-gray-700 rounded-sm my-4 p-2" type="number" name="question_%d" placeholder="Enter a number">`, q.ID)

	default:
		return fmt.Sprintf(`<input class="block text-white appearance-auto bg-gray-700 rounded-sm my-4 p-2" type="text" name="question_%d" placeholder="Enter your answer">`, q.ID)
	}
}

func ParseJson(path string) []HtmlQuestion {
	jsondata := readJson(path)
	HtmlQuestions := make([]HtmlQuestion, 0)
	for _, q := range jsondata.Questions {

		parsedQuestion := HtmlQuestion{
			Question: q,
			HTML:     generateHTML(q),
		}
		HtmlQuestions = append(HtmlQuestions, parsedQuestion)
	}
	return HtmlQuestions
}
