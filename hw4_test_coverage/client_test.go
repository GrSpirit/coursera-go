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
            Response:   `{"Error": "Cannot parse offset (strconv.Atoi: parsing "bad": invalid syntax)"}`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "-1",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Offset must be > 0"}`,
        },
        TestSearch {
            Limit:      "1",
            Offset:     "0",
            OrderField: "Name",
            OrderBy:    "-1",
            StatusCode: http.StatusOK,
            Response:   `[{"Id":15,"Name":"Allison Valdez","Age":21,"About":"Labore excepteur voluptate velit occaecat est nisi minim. Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. Ullamco sint nisi voluptate cillum nostrud aliquip et minim. Enim duis esse do aute qui officia ipsum ut occaecat deserunt. Pariatur pariatur nisi do ad dolore reprehenderit et et enim esse dolor qui. Excepteur ullamco adipisicing qui adipisicing tempor minim aliquip.\n","Gender":"male"}]`,
        },
        TestSearch {
            Limit:      "1",
            Offset:     "0",
            OrderField: "Name",
            OrderBy:    "1",
            StatusCode: http.StatusOK,
            Response:   `[{"Id":13,"Name":"Whitley Davidson","Age":40,"About":"Consectetur dolore anim veniam aliqua deserunt officia eu. Et ullamco commodo ad officia duis ex incididunt proident consequat nostrud proident quis tempor. Sunt magna ad excepteur eu sint aliqua eiusmod deserunt proident. Do labore est dolore voluptate ullamco est dolore excepteur magna duis quis. Quis laborum deserunt ipsum velit occaecat est laborum enim aute. Officia dolore sit voluptate quis mollit veniam. Laborum nisi ullamco nisi sit nulla cillum et id nisi.\n","Gender":"male"}]`,
        },
        TestSearch {
            Limit:      "1",
            Offset:     "0",
            OrderField: "Age",
            OrderBy:    "-1",
            StatusCode: http.StatusOK,
            Response:   `[{"Id":1,"Name":"Hilda Mayer","Age":21,"About":"Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n","Gender":"female"}]`,
        },
        TestSearch {
            Limit:      "1",
            Offset:     "0",
            OrderField: "Age",
            OrderBy:    "1",
            StatusCode: http.StatusOK,
            Response:   `[{"Id":32,"Name":"Christy Knapp","Age":40,"About":"Incididunt culpa dolore laborum cupidatat consequat. Aliquip cupidatat pariatur sit consectetur laboris labore anim labore. Est sint ut ipsum dolor ipsum nisi tempor in tempor aliqua. Aliquip labore cillum est consequat anim officia non reprehenderit ex duis elit. Amet aliqua eu ad velit incididunt ad ut magna. Culpa dolore qui anim consequat commodo aute.\n","Gender":"female"}]`,
        },
        TestSearch {
            Limit:      "10",
            Offset:     "0",
            OrderBy:    "bad",
            StatusCode: http.StatusBadRequest,
            Response:   `{"Error": "Cannot parse order_by (strconv.Atoi: parsing "bad": invalid syntax)"}`,
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
            Query:      "Newman",
            StatusCode: http.StatusOK,
            Response:   `[{"Id":14,"Name":"Nicholson Newman","Age":23,"About":"Tempor minim reprehenderit dolore et ad. Irure id fugiat incididunt do amet veniam ex consequat. Quis ad ipsum excepteur eiusmod mollit nulla amet velit quis duis ut irure.\n","Gender":"male"}]`,
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

type TestCase struct {
    Request     SearchRequest
    Response    SearchResponse
    Err         error
}

func (t TestCase) NextPage() bool {
    if t.Request.Limit == len(t.Response.Users) {
        return true
    }
    return false
}

func TestFindUsers(t *testing.T) {
    cases := []TestCase {
        TestCase {
            Request: SearchRequest{
                Limit: 1,
                Offset: 0,
                OrderField: "Id",
                OrderBy: -1,
            },
            Response:  SearchResponse{
                Users: []User {
                    User{
                        Id: 0,
                        Name: "Boyd Wolf",
                        Age: 22,
                        About: "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
                        Gender: "male",
                    },
                },
            },
        },
        TestCase{
            Request: SearchRequest{
                Limit:      1,
                Offset:     0,
                OrderField: "Name",
                OrderBy:    -1,
            },
            Response: SearchResponse {
                Users: []User{
                    User{
                        Id: 15,
                        Name: "Allison Valdez",
                        Age: 21,
                        About: "Labore excepteur voluptate velit occaecat est nisi minim. Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. Ullamco sint nisi voluptate cillum nostrud aliquip et minim. Enim duis esse do aute qui officia ipsum ut occaecat deserunt. Pariatur pariatur nisi do ad dolore reprehenderit et et enim esse dolor qui. Excepteur ullamco adipisicing qui adipisicing tempor minim aliquip.\n",
                        Gender: "male",
                    },
                },
            },
        },
    }

    srv := httptest.NewServer(http.HandlerFunc(SearchServer))

    for caseNum, item := range cases {
        client := &SearchClient {
            URL:    testSrv.URL,
        }
        result, err := user.FindUsers(item.Request)
        if err != nil {

            continue
        }

        if result.NextPage != item.NextPage() {
            t.Errorf("[%d] NextPage does not match: expected %v got %v", caseNum, item.NextPage(), result.NextPage)
        }
        if len(result.Users) != len(item.Response.Users) {
            t.Errorf("[%d] Invalid number of users: expected %v got %v", caseNum, len(item.Response.Users), len(result.Users))
            continue
        }

        for i, u := range result.Users {
            if u.Id != item.Response.Users[i].Id || u.Name != item.Response.Users[i].Name || a.Age != item.Response.Users[i].item.Response.Users[i].Age || u.About != item.Response.Users[i].About || u.Gender != item.Response.Users[i].Gender {
                t.Errorf("[%d] User do not match: expected $+v got %+v", caseNum, item.Response.Users[i], u)
            }
        }
    }

}
