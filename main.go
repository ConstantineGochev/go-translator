package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type TranslationRequest struct {
	EnglishWord     string `json:"english-word,omitempty"`
	EnglishSentence string `json:"english-sentence,omitempty"`
}
type TranslationResponse struct {
	GopherWord     string `json:"gopher-word,omitempty"`
	GopherSentence string `json:"gopher-sentence,omitempty"`
}

type State struct {
	Data map[string]string
	ctx  *context.Context
}

var vowels = [...]string{"a", "e", "i", "o", "u", "y"}

func translate_word(eng_w string) string {
	var t string
	var first_vowel_indx int
	temp := make(map[int]string)
	for _, v := range vowels {
		vowel_indx := strings.Index(eng_w, v)
		if vowel_indx != -1 {

			// temp = append(temp, v)
			temp[vowel_indx] = v
		}
		//log.Printf("temp len %q", len(temp))
	}
	log.Printf("temp[] %q", temp)
	var keys []int
	for k := range temp {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	first_vowel_indx = strings.Index(eng_w, temp[keys[0]])
	log.Printf(" first vowel index %q", strings.Index(eng_w, temp[keys[0]]))
	if first_vowel_indx == 0 {
		t = "g" + eng_w
		return t
	}
	if string(eng_w[0])+string(eng_w[1]) == "xr" {
		t = "ge" + eng_w
		return t
	}
	log.Printf("w with index %q", eng_w[:first_vowel_indx])
	if first_vowel_indx > 0 &&
		eng_w[:first_vowel_indx] == "u" &&
		eng_w[:first_vowel_indx-1] == "q" {
		ss := strings.SplitAfterN(eng_w, "qu", 2)
		t = ss[1] + ss[0] + "ogo"
		return t
	}
	if first_vowel_indx > 0 {
		ss := strings.SplitAfterN(eng_w, eng_w[:first_vowel_indx], 2)
		t = ss[1] + ss[0] + "ogo"
		return t
	}
	return t
}

func (s *State) post_word(w http.ResponseWriter, req *http.Request) {
	b, body_err := ioutil.ReadAll(req.Body)
	if body_err != nil {
		log.Print("body Err ", body_err.Error())
		http.Error(w, body_err.Error(), http.StatusInternalServerError)
		return
	}
	var t TranslationRequest
	err := json.Unmarshal(b, &t)
	if err != nil {
		fmt.Println(err)
		return
	}
	var resp TranslationResponse = TranslationResponse{GopherWord: translate_word(t.EnglishWord)}
	s.Data[t.EnglishWord] = resp.GopherWord
	log.Printf("T %q", resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
func (s *State) post_sentence(w http.ResponseWriter, req *http.Request) {

	b, body_err := ioutil.ReadAll(req.Body)
	if body_err != nil {
		log.Print("body Err ", body_err.Error())
		http.Error(w, body_err.Error(), http.StatusInternalServerError)
		return
	}
	var t TranslationRequest
	err := json.Unmarshal(b, &t)
	if err != nil {
		fmt.Println(err)
		return
	}
	last_char := t.EnglishSentence[len(t.EnglishSentence)-1:]
	last_char_rune := []rune(last_char)
	if unicode.IsPunct(last_char_rune[0]) {
		log.Println("lastChar is a punctuation :", last_char)
		t.EnglishSentence = strings.TrimSuffix(t.EnglishSentence, last_char)
	}
	words := strings.Fields(t.EnglishSentence)
	var ta []string
	for _, w := range words {
		translated := translate_word(w)
		ta = append(ta, translated)
	}
	ta_str := strings.Join(ta, " ")
	log.Printf("TA %q", ta_str)
	if unicode.IsPunct(last_char_rune[0]) {
		ta_str = ta_str + last_char
	}

	var resp TranslationResponse = TranslationResponse{GopherSentence: ta_str}

	s.Data[t.EnglishSentence] = resp.GopherSentence
	log.Printf("T %q", resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

type CustomMap map[string]string

func (sa CustomMap) MarshalJSON() ([]byte, error) {
	type InObj map[string]string
	var out struct {
		Map []InObj `json:"history"`
	}
	for k, v := range sa {
		obj := make(InObj)
		obj[k] = v
		out.Map = append(out.Map, obj)

	}
	return json.Marshal(out)
}
func (s *State) history(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	keys := make([]string, 0, len(s.Data))
	sorted_data := make(CustomMap)
	log.Printf("s.Data %q", s.Data)
	for k := range s.Data {

		log.Printf("K %q", k)
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, c := range keys {
		sorted_data[c] = s.Data[c]
	}
	log.Printf("SORTEDDATA %q", sorted_data)
	j, err := json.Marshal(sorted_data)
	if err != nil {
		log.Print("JSON Error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}
func main() {
	var port int
	flag.IntVar(&port, "p", 8000, "specify port to use.  defaults to 8000.")
	flag.Parse()
	fmt.Printf("port = %d", port)
	c := context.TODO()
	s := &State{ctx: &c, Data: map[string]string{}}
	http.HandleFunc("/word", s.post_word)
	http.HandleFunc("/sentence", s.post_sentence)
	http.HandleFunc("/history", s.history)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}
