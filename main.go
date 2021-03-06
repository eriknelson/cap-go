package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"

	//"github.com/codeskyblue/go-sh"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const MAIN_FILE = "Nulecule"

// TODO: create a struct
type Answers map[string]map[string]string

func runCommand(cmd string, args ...string) []byte {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		fmt.Println("Error running " + cmd)
	}
	return output
}

// returns a map of maps
func parseBasicINI(data string) map[string]map[string]string {
	/*
		find first [ then find matching ]. Everything between them is the first key. Read until next [ or end of string.
	*/
	var answers = make(map[string]map[string]string)
	values := strings.SplitAfter(data, "\n")
	var key string
	for _, str := range values {
		if strings.HasPrefix(str, "[") {
			key = strings.Trim(str, "[]\n")
			answers[key] = make(map[string]string)
		} else {
			subvalue := strings.Split(str, " = ")
			answers[key][subvalue[0]] = strings.Trim(subvalue[1], "\n")
		}
	}

	fmt.Println(answers)
	return answers
}

func getAnswersFromFile(nulecule_path string) map[string]Answers {
	os.Remove("answers.conf")
	/*
		output, err := exec.Command("atomicapp", "genanswers", "nulecule-library/"+nulecule_path).CombinedOutput()
		if err != nil {
			fmt.Println("Error running atomicapp")
		}
	*/
	output := runCommand("atomicapp", "genanswers", "nulecule-library/"+nulecule_path)
	fmt.Println(string(output))
	answers, err := ioutil.ReadFile("answers.conf")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(answers))
	// add root node
	return map[string]Answers{"nulecule": parseBasicINI(string(answers))}
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
	json.NewEncoder(w).Encode(getNuleculeList())
}

func NuleculeDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nulecule_id := vars["id"]
	json.NewEncoder(w).Encode(getAnswersFromFile(nulecule_id))
}

func genUUID() string {
	return strings.Trim(string(runCommand("/usr/bin/uuidgen")), "\n")
}

func getToken() string {
	return strings.Trim(string(runCommand("/usr/bin/oc", "whoami", "-t")), "\n")
}

func createNewProject(project string) string {
	return strings.Trim(string(runCommand("/usr/bin/oc", "new-project", project)), "\n")
}

func addProviderDetails(answers Answers) {
	uuid := genUUID()
	token := getToken()
	project_name := "cap-" + uuid
	output := createNewProject(project_name)
	fmt.Println(output)
	provider := make(map[string]string)
	provider["namespace"] = project_name
	provider["provider"] = "openshift"
	provider["provider-api"] = "https://10.1.2.2:8443"
	provider["provider-auth"] = token
	provider["provider-cafile"] = "/host/var/lib/openshift/openshift.local.config/master/ca.crt"
	provider["providertlsverify"] = "False"
	answers["general"] = provider
}

func NuleculeUpdate(w http.ResponseWriter, r *http.Request) {
	// update the nulecule answers file
	vars := mux.Vars(r)
	nulecule_id := vars["id"]
	fmt.Println(nulecule_id)
	fmt.Println("NuleculeUpdate!")

	// get the posted answers
	// Answers is a map of maps
	res_map := make(map[string]Answers)

	json.NewDecoder(r.Body).Decode(&res_map)

	// ERIK TODO:
	// -> Convert answer JSON params -> map[string]interface{}
	// -> answerMap := addProviderDetails(map[string]interface{}) < adds provider necessary details to [general]
	// -> iniStruct := genINIFromAnswers(answerMap)
	// -> iniStruct.write(/* target nulecule directory */
	addProviderDetails(res_map["nulecule"])

	home_dir := getHomeDir()
	answers_dir := path.Join(home_dir, "answers", nulecule_id)
	os.MkdirAll(answers_dir, 0755)

	f, err := os.Create(path.Join(answers_dir, "answers.conf"))
	if err != nil {
		fmt.Println("Error creating answers.conf")
	}

	defer f.Close()

	for k, v := range res_map["nulecule"] {
		//fmt.Print("[" + k + "]\n")
		fmt.Fprint(f, "["+k+"]\n")
		for k1, v1 := range v {
			//fmt.Printf("%s=%s\n", k1, v1)
			fmt.Fprintf(f, "%s=%s\n", k1, v1)
		}
	}

	json.NewEncoder(w).Encode(res_map) // Success, fail?
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

func NuleculeDeploy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nulecule_id := vars["id"]

	home_dir := getHomeDir()

	// Create nulecules dir if it doens't already exist
	nulecules_dir := path.Join(home_dir, "nulecules")
	mode := os.FileMode(int(0755))
	os.Mkdir(nulecules_dir, mode)

	nulecule_dir := path.Join(nulecules_dir, nulecule_id)

	// Download atomicapp
	download_script := path.Join(mainGoDir(), "download_atomicapp.sh")
	output := runCommand("bash", download_script, nulecule_id)
	fmt.Println(string(output))

	// Fix the fact that the entire thing is owned by root -.- WHY
	output = runCommand(
		"sudo", "chown", "-R", "vagrant:vagrant", nulecule_dir)
	fmt.Println(string(output))

	// Copy in generated answers.conf from $HOME/answers working directory
	answers_conf_src := path.Join(home_dir, "answers", nulecule_id, "answers.conf")
	output = runCommand("cp", answers_conf_src, nulecule_dir)
	fmt.Println(string(output))

	// Run the atomicapp!
	run_script := path.Join(mainGoDir(), "run_atomicapp.sh")
	output = runCommand("bash", run_script, nulecule_id)
	fmt.Println(string(output))

	// TODO: EXPOSE ROUTE!
	// Need to figure out a way to tie the "svc" that was just
	// created with the atomicapp that was deployed so we can
	// expose the route correctly.
	//
	// `oc get svc`
	// `oc expose service etherpad-svc -l name=etherpad`

	// TODO: Error handling!
	res_map := make(map[string]interface{})
	res_map["result"] = "success"

	json.NewEncoder(w).Encode(res_map) // Success, fail?
}

func wrapScriptCmd(cmd string) string {
	return fmt.Sprintf("\"%s\"", cmd)
}

func mainGoDir() string {
	/*
		_, filename, _, _ := runtime.Caller(0)
		return fmt.Sprintf(path.Dir(filename))
	*/
	return "."
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/nulecules", Nulecules)
	r.HandleFunc("/nulecules/{id}", NuleculeDetails).Methods("GET")
	r.HandleFunc("/nulecules/{id}", NuleculeUpdate).Methods("POST")
	r.HandleFunc("/nulecules/{id}/deploy", NuleculeDeploy).Methods("POST")
	fmt.Println("Listening on localhost:3001")

	allowed_headers := handlers.AllowedHeaders([]string{"Content-Type"})

	log.Fatal(http.ListenAndServe(":3001", handlers.CORS(
		allowed_headers,
	)(r)))
}
