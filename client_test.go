// client_test.go
package main

//TODO search("hn Jo") -> "John Johnson" ??? search("on Jo") -> "Johnson John" ???
//TODO cache search results that don't fit in current page (NextPage==true)
//TODO sort: []*User faster than []User ???
//TODO case insensitive search ???
//TODO sort: use clojures ???

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type UserData struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"` // \_ Name field
	LastName  string `xml:"last_name"`  // /  of User
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type UsersData struct {
	Version string     `xml:"version,attr"`
	List    []UserData `xml:"row"`
}

type TestResult struct {
	response *SearchResponse
	err      error
}

type TestCase struct {
	SClient  *SearchClient
	SRequest SearchRequest
	Result   TestResult
}

// --- for sorting:
type Users []User

func (s Users) Len() int      { return len(s) }
func (s Users) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// --- sort by Id:
type ById struct{ Users }

func (s ById) Less(i, j int) bool { return s.Users[i].Id < s.Users[j].Id }

// --- sort by Age:
type ByAge struct{ Users }

func (s ByAge) Less(i, j int) bool { return s.Users[i].Age < s.Users[j].Age }

// --- sort by Name:
type ByName struct{ Users }

func (s ByName) Less(i, j int) bool { return s.Users[i].Name < s.Users[j].Name }

// --- sort by Id:
type ByIdDesc struct{ Users }

func (s ByIdDesc) Less(i, j int) bool { return s.Users[i].Id > s.Users[j].Id }

// --- sort by Age:
type ByAgeDesc struct{ Users }

func (s ByAgeDesc) Less(i, j int) bool { return s.Users[i].Age > s.Users[j].Age }

// --- sort by Name:
type ByNameDesc struct{ Users }

func (s ByNameDesc) Less(i, j int) bool { return s.Users[i].Name > s.Users[j].Name }

//TODO use json.Marshal()
const jsonStr0 = `[
	{"Id": "%v", "Name": "%v", "Age": "%v", "About": "%v", "Gender": "%v"},
]`

//TODO
// 0) auth(atoken)
// 1) query
// 2) sort(order_field, order_by)
// 3) slice(offset, limit)
func SearchServer(w http.ResponseWriter, r *http.Request) {
	//- authorization(atoken) --------------------------------------------------
	//fmt.Println(r.Header)
	atoken := r.Header.Get("AccessToken")
	fmt.Println("-- atoken", atoken)
	if atoken != "1234" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	//--------------------------------------------------------------------------

	// empty result: ?TODO don't process explicitly?
	limit, err := strconv.Atoi(r.FormValue("limit"))
	fmt.Println("-- limit:", limit)
	if err != nil || limit < 0 { //TODO ??? combine StatusBadRequest-conditions ???
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	offset, err := strconv.Atoi(r.FormValue("offset"))
	fmt.Println("-- offset:", offset)
	if err != nil || offset < 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	query := r.FormValue("query")
	fmt.Println("-- query:", query)
	//TODO ??? check

	order_field := r.FormValue("order_field")
	fmt.Println("-- order_field:", order_field)
	if order_field != "" && order_field != "Id" && order_field != "Age" && order_field != "Name" { //TODO check later by switch/case?
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	order_by, err := strconv.Atoi(r.FormValue("order_by"))
	fmt.Println("-- order_by:", order_by)
	if err != nil || order_by < -1 || order_by > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	xmlData, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) //???
		return
	}

	usersData := new(UsersData)
	xml.Unmarshal(xmlData, &usersData)
	//	fmt.Println("=== 1st client:", usersData.List[0])
	//	fmt.Println("=== 2nd client:", usersData.List[1])

	//TODO query=="" -> can be optimized ???
	users := Users{} //TODO make(capacity>0) ???
	for _, ud := range usersData.List {
		if query == "" || strings.Contains(ud.FirstName, query) || strings.Contains(ud.LastName, query) || strings.Contains(ud.About, query) {
			users = append(users, User{
				Id:     ud.Id,
				Name:   ud.FirstName + " " + ud.LastName, //TODO ??? " "
				Age:    ud.Age,
				About:  ud.About,
				Gender: ud.Gender,
			})
		}
	}

	// sorting:
	if order_by == 1 {
		switch order_field {
		case "Id":
			sort.Sort(ById{users})
		case "Age":
			sort.Sort(ByAge{users})
		case "", "Name":
			sort.Sort(ByName{users})
		default: //checked before
		}
	} else if order_by == -1 {
		switch order_field {
		case "Id":
			sort.Sort(ByIdDesc{users})
		case "Age":
			sort.Sort(ByAgeDesc{users})
		case "", "Name":
			sort.Sort(ByNameDesc{users})
		default: //checked before
		}
	}

	// slicing(limit, offset):
	if limit < len(users) || offset > 0 { //TODO ??? +-1 !!!
		users = users[offset:limit]
	}

	//	users := []User{
	//		User{
	//			Id:     0,
	//			Name:   "John Johnson",
	//			Age:    33,
	//			About:  "Lorem ipsum",
	//			Gender: "male",
	//		},
	//	}
	//	users := []User{}

	for _, u := range users {
		fmt.Println("-u-", u)
	}

	usersJSON, err := json.Marshal(users)
	if err == nil {
		fmt.Println("-=-", string(usersJSON))
		fmt.Fprintf(w, string(usersJSON))
	}
}

func TestFindUsers(t *testing.T) {
	//TODO init...
	cases := []TestCase{
		TestCase{
			SClient: &SearchClient{
				AccessToken: "1234",
				//URL:         "http://127.0.0.1:8080",
			},
			SRequest: SearchRequest{
				Limit:      26, //0 //1 //26
				Offset:     0,
				Query:      "minim", //An", //"John",
				OrderField: "Id",    //"Name",
				OrderBy:    -1,
			},
			Result: TestResult{
				response: &SearchResponse{
					Users: []User{
						User{
							Id:     0,
							Name:   "John Johnson",
							Age:    33,
							About:  "Lorem ipsum",
							Gender: "male",
						},
					},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	var result TestResult
	for caseNum, item := range cases {
		item.SClient.URL = ts.URL //!!! ??? TODO vary for coverage ???
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected (%#v, %#v), got (%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}

		//		if !reflect.DeepEqual(item.Result.response, result.response) {
		//			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result.response, result.response)
		//		}
		//		if !reflect.DeepEqual(item.Result.err, result.err) {
		//			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result.err, result.err)
		//		}
	}
}
