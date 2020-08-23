package gsd

import (
	"html/template"
	"io/ioutil"
)

func readTemplate(name string) *template.Template {

	data, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}

	t, err := template.New(name).Parse(string(data))
	if err != nil {
		panic(err)
	}

	return t
}

func readTemplates() {

}
