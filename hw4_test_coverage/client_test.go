package main

import (
    "strconv"
    "net/http"
    "net/http/httprequest"
)

// код писать тут
func ParseParams(r *http.Request) (SearchRequest, error) {
    var limit, offset, orderBy int
    var query, orderField string
    if r.FormValue("limit") != "" {
        limit, err := strconv.Atoi(r.FormValue("limit"))
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
        offset, err := strconv.Atoi(r.FormValue("offset"))
        if err != nil {
            return SearchRequest{}, fmt.Errorf("Cannot parse offset (%v)", err)
        }
        if offset < 0 {
            return SearchRequest{}, fmt.Errorf("Offset must be > 0")
        }
    }
    if r.FormValue("order_by") != "" {
        orderBy, err := strconv.Atoi(r.FormValue("order_by"))
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
        OrderBy: orderBy
    }, nil
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
    params, err := ParseParams(r)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        io.WriteString(w, fmt.Printf(`{"Error": "%v"}`, err))
        return
    }
}
