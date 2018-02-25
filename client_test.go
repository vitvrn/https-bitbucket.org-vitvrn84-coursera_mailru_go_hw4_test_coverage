// client_test.go
package main

//TODO search("hn Jo") -> "John Johnson" ??? search("on Jo") -> "Johnson John" ???
//TODO cache search results that don't fit in current page (NextPage==true)
//TODO sort: []*User faster than []User ???
//TODO case insensitive search ???
//TODO sort: use clojures ???
//TODO case(query=="") can be optimized ???
//TODO refactor repeating into func Test ???

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
	"time"
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

const (
	testAccessTokenGood = "1234"
	testAccessTokenBad  = ""
)

func accessTokenOk(token string) bool {
	return token == testAccessTokenGood
}

//=============================================================================
//=============================================================================
func SearchServer(w http.ResponseWriter, r *http.Request) {
	// authorization(atoken)
	atoken := r.Header.Get("AccessToken")
	fmt.Println("-- atoken", atoken)
	if !accessTokenOk(atoken) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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
	fmt.Println("-- query:", query) //TODO ??? check

	order_field := r.FormValue("order_field")
	fmt.Println("-- order_field:", order_field)
	if order_field != "" && order_field != "Id" && order_field != "Age" && order_field != "Name" { //TODO check later by switch/case?
		errorResp := SearchErrorResponse{Error: "ErrorBadOrderField"}
		errorRespjson, err := json.Marshal(&errorResp) //& ???
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError) //panic?
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorRespjson)
		return
	}

	order_by, err := strconv.Atoi(r.FormValue("order_by"))
	fmt.Println("-- order_by:", order_by)
	if err != nil || order_by < -1 || order_by > 1 {
		errorResp := SearchErrorResponse{Error: "ErrorBadOrderBy"}
		errorRespjson, err := json.Marshal(&errorResp) //& ???
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError) //panic?
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorRespjson)
		return
	}

	// loading data
	xmlData, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) //???
		return
	}

	usersData := new(UsersData)
	xml.Unmarshal(xmlData, &usersData)

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

	usersJSON, err := json.Marshal(users)
	if err == nil {
		fmt.Println("-=-", string(usersJSON))
		fmt.Fprintf(w, string(usersJSON))
	}
}

//=============================================================================
//=============================================================================

//waits for timeout to happen
func TimeoutServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(1001 * time.Millisecond)
}

func InternalErrorServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func BadErrorJsonServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("[")) //error JSON
}

func BadResultJsonServer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("[")) //error JSON
}

//=============================================================================
/*
func _TestFindUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	cases := []TestCase{
		TestCase{
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
				URL:         ts.URL,
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

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}
*/

//=============================================================================
// this tests don't need SearchServer:
//limit<0, limit>25
//offset<0
//unknown eror (invalid URL)
func TestFindUsersErrors1(t *testing.T) {
	cases := []TestCase{
		TestCase{ //limit<0
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
			},
			SRequest: SearchRequest{
				Limit: -1,
			},
			Result: TestResult{
				err: fmt.Errorf("limit must be > 0"),
			},
		},
		TestCase{ //offset<0
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
			},
			SRequest: SearchRequest{
				Offset: -1,
			},
			Result: TestResult{
				err: fmt.Errorf("offset must be > 0"),
			},
		},
		TestCase{ //limit>25 (,offset<0)
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
			},
			SRequest: SearchRequest{
				Limit:  26,
				Offset: -1,
			},
			Result: TestResult{
				err: fmt.Errorf("offset must be > 0"),
			},
		},
		TestCase{
			SClient: &SearchClient{
				URL: "1",
			},
			SRequest: SearchRequest{},
			Result: TestResult{
				err: fmt.Errorf("unknown error Get 1?limit=1&offset=0&order_by=0&order_field=&query=: unsupported protocol scheme \"\""), //??????
			},
		},
	}

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}

//=============================================================================
//T(err) == net.Error, err.Timeout() == true
func TestFindUsersErrorTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(TimeoutServer))
	defer ts.Close()

	cases := []TestCase{
		TestCase{
			SClient: &SearchClient{
				URL: ts.URL,
			},
			SRequest: SearchRequest{},
			Result: TestResult{
				err: fmt.Errorf("timeout for limit=1&offset=0&order_by=0&order_field=&query="),
			},
		},
	}

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}

//=============================================================================
//internal SearchServer error
func TestFindUsersInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(InternalErrorServer))
	defer ts.Close()

	cases := []TestCase{
		TestCase{
			SClient: &SearchClient{
				URL: ts.URL,
			},
			SRequest: SearchRequest{},
			Result: TestResult{
				err: fmt.Errorf("SearchServer fatal error"),
			},
		},
	}

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}

//=============================================================================
//BadRequest: internal server error bad error JSON
func TestFindUsersInternalServerErrorBadErrorJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(BadErrorJsonServer))
	defer ts.Close()

	cases := []TestCase{
		TestCase{
			SClient: &SearchClient{
				URL: ts.URL,
			},
			SRequest: SearchRequest{},
			Result: TestResult{
				err: fmt.Errorf("cant unpack error json: unexpected end of JSON input"),
			},
		},
	}

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}

//=============================================================================
//StatusOK: bad result JSON
func TestFindUsersInternalServerErrorBadResultJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(BadResultJsonServer))
	defer ts.Close()

	cases := []TestCase{
		TestCase{
			SClient: &SearchClient{
				URL: ts.URL,
			},
			SRequest: SearchRequest{},
			Result: TestResult{
				err: fmt.Errorf("cant unpack result json: unexpected end of JSON input"),
			},
		},
	}

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}

//=============================================================================
//unauthorized
//BadRequest: bad order field
//BadRequest: bad order by
//len(data)==req.Limit
func TestFindUsersErrors2(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	cases := []TestCase{ //unauthorized
		TestCase{
			SClient: &SearchClient{
				AccessToken: testAccessTokenBad,
				URL:         ts.URL,
			},
			SRequest: SearchRequest{},
			Result: TestResult{
				err: fmt.Errorf("Bad AccessToken"),
			},
		},
		TestCase{ //BadRequest: bad order field
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
				URL:         ts.URL,
			},
			SRequest: SearchRequest{
				OrderField: "Gender",
			},
			Result: TestResult{
				err: fmt.Errorf("OrderFeld Gender invalid"),
			},
		},
		TestCase{ //BadRequest: bad OrderBy
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
				URL:         ts.URL,
			},
			SRequest: SearchRequest{
				OrderBy: 2,
			},
			Result: TestResult{
				err: fmt.Errorf("unknown bad request error: ErrorBadOrderBy"),
			},
		},
		TestCase{ //len(data)==req.Limit
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
				URL:         ts.URL,
			},
			SRequest: SearchRequest{
				Query: "Wolf", //Limit: 0
			},
			Result: TestResult{
				response: &SearchResponse{
					Users:    []User{},
					NextPage: true,
				},
			},
		},
		TestCase{ //len(data)!=req.Limit
			SClient: &SearchClient{
				AccessToken: testAccessTokenGood,
				URL:         ts.URL,
			},
			SRequest: SearchRequest{
				Limit: 1,
				Query: "Wolf",
			},
			Result: TestResult{
				response: &SearchResponse{
					Users: []User{
						User{
							Id:     0,
							Name:   "Boyd Wolf",
							Age:    22,
							About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
							Gender: "male",
						},
					},
				},
			},
		},
	}

	var result TestResult
	for caseNum, item := range cases {
		result.response, result.err = item.SClient.FindUsers(item.SRequest)

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("\n[%d] wrong result, expected \n(%#v, %#v) \ngot \n(%#v, %#v)", caseNum, item.Result.response, item.Result.err, result.response, result.err)
		}
	}
}
