// client_test.go
package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type Client struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"` // \_ Name field
	LastName  string `xml:"last_name"`  // /  of User
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Clients struct {
	Version string   `xml:"version,attr"`
	List    []Client `xml:"row"`
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

//TODO use json.Marshal()
const jsonStr0 = `[
	{"Id": "%v", "Name": "%v", "Age": "%v", "About": "%v", "Gender": "%v"},
]`

func SearchServer(w http.ResponseWriter, r *http.Request) {
	limit := r.FormValue("limit")
	offset := r.FormValue("offset")
	query := r.FormValue("query")
	order_field := r.FormValue("order_field")
	order_by := r.FormValue("order_by")

	atoken, ok := r.Header["AccessToken"]
	//TODO ...
	fmt.Println(limit)
	fmt.Println(offset)
	fmt.Println(query)
	fmt.Println(order_field)
	fmt.Println(order_by)
	fmt.Println(atoken, ok)

	xmlData, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		panic("Error: cannot read the xml file")
	}
	clients := new(Clients)
	xml.Unmarshal(xmlData, &clients)
	fmt.Println("=== 1st client:", clients.List[0])
	fmt.Println("=== 2nd client:", clients.List[1])

	//TODO debug, del:
	users := []User{
		User{
			Id:     0,
			Name:   "John Johnson",
			Age:    33,
			About:  "Lorem ipsum",
			Gender: "male",
		},
	}
	usersJSON, err := json.Marshal(users)
	if err == nil {
		fmt.Println(string(usersJSON))
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
				Limit:      1,
				Offset:     0,
				Query:      "John",
				OrderField: "Name",
				OrderBy:    0,
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
	//TODO for tests...
	var result TestResult
	for caseNum, item := range cases {
		item.SClient.URL = ts.URL //!!!
		result.response, result.err = item.SClient.FindUsers(item.SRequest)
		//		if !reflect.DeepEqual(item.Result, result) {
		//			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		//		}
		if !reflect.DeepEqual(item.Result.response, result.response) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result.response, result.response)
		}
		if !reflect.DeepEqual(item.Result.err, result.err) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result.err, result.err)
		}
	}
	//TODO check results...
}
