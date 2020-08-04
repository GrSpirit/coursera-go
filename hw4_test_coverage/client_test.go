package main

import (
    "fmt"
    "io"
    "os"
    "io/ioutil"
    "sort"
    "strings"
    "strconv"
    "net/url"
    "net/http"
    "net/http/httptest"
    "testing"
    "encoding/xml"
    "encoding/json"
)

type RowXml struct {
    Id          int     `xml:"id"`
    FirstName   string  `xml:"first_name"`
    LastName    string  `xml:"last_name"`
    Age         int     `xml:"age"`
    About       string  `xml:"about"`
    Gender      string  `xml:"gender"`
}

type RootXml struct {
    Rows        []RowXml    `xml:"row"`
}

type ServerConfig struct {
    BaseURL     string
}

var config = ServerConfig {
    BaseURL:    "http://localhost:8080/",
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func ParseParams(r *http.Request) (SearchRequest, error) {
    var err error
    var limit, offset, orderBy int
    var query, orderField string
    if r.FormValue("limit") != "" {
        limit, err = strconv.Atoi(r.FormValue("limit"))
        if err != nil {
            return SearchRequest{}, fmt.Errorf("Cannot parse limit (%v)", err)
        }
        if limit < 0 {
            return SearchRequest{}, fmt.Errorf("Limit must be > 0")
        }
        if limit > 25 {
            limit = 25
        }
    }
    if r.FormValue("offset") != "" {
        offset, err = strconv.Atoi(r.FormValue("offset"))
        if err != nil {
            return SearchRequest{}, fmt.Errorf("Cannot parse offset (%v)", err)
        }
        if offset < 0 {
            return SearchRequest{}, fmt.Errorf("Offset must be > 0")
        }
    }
    if r.FormValue("order_by") != "" {
        orderBy, err = strconv.Atoi(r.FormValue("order_by"))
        if err != nil {
            return SearchRequest{}, fmt.Errorf("Cannot parse order_by (%v)", err)
        }
        if orderBy < -1 || orderBy > 1 {
            return SearchRequest{}, fmt.Errorf("OrderBy invalid")
        }
    }
    query = r.FormValue("query")
    orderField = r.FormValue("order_field")
    if orderField == "" {
        orderField = "Name"
    } else if orderField != "Name" && orderField != "Id" && orderField != "Age" {
        return SearchRequest{}, fmt.Errorf("ErrorBadOrderField")
    }
    return SearchRequest {
        Limit: limit,
        Offset: offset,
        Query: query,
        OrderField: orderField,
        OrderBy: orderBy,
    }, nil
}

func ConvertUsersXml(users []RowXml) []User {
    result := make([]User, 0, len(users))
    for _, u := range users {
        result = append(result, User {
            Id:     u.Id,
            Name:   u.FirstName + " " + u.LastName,
            Age:    u.Age,
            Gender: u.Gender,
            About:  u.About,
        })
    }
    return result
}

func FilterUsers(users []User, filter string) []User {
    newUsers := make([]User, 0)
    for _, u := range users {
        if strings.Contains(u.Name, filter) {
            newUsers = append(newUsers, u)
        }
    }
    return newUsers
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
    params, err := ParseParams(r)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        io.WriteString(w, fmt.Sprintf(`{"Error": "%v"}`, err))
        return
    }
    fmt.Printf("%+v\n", params)
    file, err := os.Open("dataset.xml")
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        io.WriteString(w, fmt.Sprintf(`{"Error": "Cannot open dataset"}`))
        return
    }
    defer file.Close()
    data, err := ioutil.ReadAll(file)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        io.WriteString(w, fmt.Sprintf(`{"Error": "Cannot read dataset"}`))
        return
    }

    rootXml := RootXml{}
    err = xml.Unmarshal(data, &rootXml)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        io.WriteString(w, fmt.Sprintf(`{"Error": "Cannot parse dataset"}`))
        return
    }
    fmt.Printf("Read users: %v\n", len(rootXml.Rows))

    users := ConvertUsersXml(rootXml.Rows)

    if params.Query != "" {
        users = FilterUsers(users, params.Query)
    }

    if params.OrderField != "" {
        var order = func(orderField string, orderBy int) func(i, j int) bool {
            cmpInt := map[int]func(a, b int) bool {
                -1: func(a, b int) bool { return a < b },
                0:  func(_, _ int) bool { return true },
                1:  func(a, b int) bool { return a > b },
            }
            cmpStr := map[int]func(a, b string) bool {
                -1: func(a, b string) bool { return a < b },
                0:  func(_, _ string) bool { return true },
                1:  func(a, b string) bool { return a > b },
            }
            field := map[string] func(i, j int) bool {
                "Id":   func(i, j int) bool {
                    return cmpInt[orderBy](users[i].Id, users[j].Id)
                },
                "Name": func(i, j int) bool {
                    return cmpStr[orderBy](users[i].Name, users[j].Name)
                },
                "Age":  func(i, j int) bool {
                    return cmpInt[orderBy](users[i].Age, users[j].Age)
                },
            }
            return field[orderField]
        }
        sort.Slice(users, order(params.OrderField, params.OrderBy))
    }

    if params.Offset > 0 {
        if params.Offset >= len(users) {
            users = users[:0]
        } else {
            users = users[params.Offset:]
        }
    }

    if params.Limit > 0 {
        users = users[:min(len(users), params.Limit)]
    }

    data, err = json.Marshal(users)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        io.WriteString(w, fmt.Sprintf(`{"Error": "Cannot serialize users"}`))
        return
    }
    w.Write(data)
    w.WriteHeader(http.StatusOK)
}

type TestSearch struct {
    AccessToken string
    Limit       string
    Offset      string
    Query       string
    OrderField  string
    OrderBy     string
    Response    string
    StatusCode  int
}

func TestSearchServer(t *testing.T) {
    cases := []TestSearch {
        TestSearch {
            Limit:      "1",
            Offset:     "0",
            OrderField: "Id",
            OrderBy:    "-1",
            StatusCode: http.StatusOK,
            Response:   `[{"Id":0,"Name":"Boyd Wolf","Age":22,"About":"Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n","Gender":"male"}]`,
        },
        TestSearch {
            Limit:      "bad",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Cannot parse limit (strconv.Atoi: parsing "bad": invalid syntax)"}`,
        },
        TestSearch {
            Limit:      "-1",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Limit must be > 0"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "bad",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Cannot parse offset"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "-1",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Offset must be > 0"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "0",
            OrderBy:    "bad",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Cannot parse order_by"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "0",
            OrderBy:    "2",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "OrderBy invalid"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "0",
            OrderBy:    "2",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "OrderBy invalid"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "0",
            Query:      "Name",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "OrderBy invalid"}`,
        },
    }

    //testServ := httptest.NewServer(http.HandlerFunc(SearchServer))
    //client := &http.Client(Timeout: time.Second)

    for caseNum, item := range cases {
        params := url.Values{}
        params.Add("limit", item.Limit)
        params.Add("offset", item.Offset)
        params.Add("query", item.Query)
        params.Add("order_field", item.OrderField)
        params.Add("order_by", item.OrderBy)

        req := httptest.NewRequest("GET", "http://localhost:8080/?" + params.Encode(), nil)
        w := httptest.NewRecorder()

        if item.AccessToken != "" {
            req.Header.Add("AccessToken", item.AccessToken)
        }

        SearchServer(w, req)

        if w.Code != item.StatusCode {
            t.Errorf("[%d] wrong StatusCode: got %d, expected %d", caseNum, w.Code, item.StatusCode)
        }

        resp := w.Result()
        body, _ := ioutil.ReadAll(resp.Body)
        bodyStr := string(body)
        if bodyStr != item.Response {
            t.Errorf("[%d] wrong Response: got\n%+v, expected\n%+v", caseNum, bodyStr, item.Response)
        }

    }

}
