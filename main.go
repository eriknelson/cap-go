package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	//"github.com/cdrage/atomicapp-go/nulecule"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func getAnswers(nulecule_path string) map[string]interface{} {
	//func getAnswers(nulecule_path string) map[string]nulecule.Answers {
	GEN_ANSWERS_SCRIPT := "/home/ernelson/cap/cap_ui/gen_answerfile.sh"
	fmt.Println("script -> %s", GEN_ANSWERS_SCRIPT)

	fmt.Println("path: " + nulecule_path)
	//base := nulecule.New("nulecule-library/"+nulecule_path, "", false)

	//err := base.ReadMainFile()
	//if err != nil {
	//fmt.Println("error reading nulecule", err)
	//}

	//err = base.LoadAnswers()
	//if err != nil {
	//fmt.Println("error loading answerse", err)
	//}

	//j, _ := json.Marshal(base)

	cmd := exec.Command(GEN_ANSWERS_SCRIPT, nulecule_path)
	output, _ := cmd.CombinedOutput()

	// TODO: Fix the hackery going on here. We're taking the JSON
	// string out of the python script, decoding it, and re-encoding
	// the data structure since go seems to be double escaping thigns
	// if I serve the string back directly. Shouldn't have to do this.

	var dat map[string]interface{}

	//dat := map[string]string{
	//"test": "bar",
	//"foo":  "bar",
	//}

	if err := json.Unmarshal(output, &dat); err != nil {
		panic(err)
	}

	return dat
}

func getNuleculeList() map[string][]string {
	files, _ := ioutil.ReadDir("./nulecule-library")
	nulecules := make([]string, 0)
	for _, f := range files {
		if f.IsDir() {
			nulecules = append(nulecules, f.Name())
		}
	}
	return map[string][]string{"nulecules": nulecules}
}

func Nulecules(w http.ResponseWriter, r *http.Request) {
	//w.Write([]byte("Gorilla!\n"))

	json.NewEncoder(w).Encode(getNuleculeList())
}

func NuleculeDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nulecule_id := vars["id"]

	res_map := make(map[string]interface{})
	res_map["nulecule"] = getAnswers(nulecule_id)
	json.NewEncoder(w).Encode(res_map)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/nulecules", Nulecules)
	r.HandleFunc("/nulecules/{id}", NuleculeDetails)
	fmt.Println("Listening on localhost:3001")
	log.Fatal(http.ListenAndServe(":3001", handlers.CORS()(r)))
}
