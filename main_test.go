package main

import (
	"fmt"
	"os"
	"testing"

	// tom: for ensureTableExists
	"log"

	"bytes"
	"encoding/json"
	// tom: for TestEmptyTable and next functions (no go get is required"
	"net/http"
	// "net/url"
	"net/http/httptest"
	"strconv"
	// "io/ioutil"
)

var a App

func TestMain(m *testing.M) {
	a = App{}
	a.Initialize(
		os.Getenv("TEST_DB_USERNAME"),
		os.Getenv("TEST_DB_PASSWORD"),
		os.Getenv("TEST_DB_NAME"))

	ensureTableExists()

	code := m.Run()

	clearTable()

	os.Exit(code)
}

func ensureTableExists() {
	if _, err := a.DB.Exec(productTableCreationQuery); err != nil {
		log.Fatal(err)
	}

	if _, err := a.DB.Exec(storeTableCreationQuery); err != nil {
		log.Fatal(err)
	}

	if _, err := a.DB.Exec(storeProductsTableCreationQuery); err != nil {
		log.Fatal(err)
	}

}

func clearTable() {
	a.DB.Exec("TRUNCATE TABLE products CASCADE")
	a.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")

	a.DB.Exec("TRUNCATE TABLE stores CASCADE")
	a.DB.Exec("ALTER SEQUENCE stores_id_seq RESTART WITH 1")

	a.DB.Exec("TRUNCATE TABLE storeProducts CASCADE")
}

const productTableCreationQuery = `

CREATE TABLE IF NOT EXISTS products
(
    id SERIAL,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT products_pkey PRIMARY KEY (id)
    
)
`

const storeTableCreationQuery = `
CREATE TABLE IF NOT EXISTS stores
(
    store_id SERIAL,
    name TEXT NOT NULL
)
`

const storeProductsTableCreationQuery = `
CREATE TABLE IF NOT EXISTS storeProducts
(	store_id int,
	product_id int,
	is_available bool
)
`

// tom: next functions added later, these require more modules: net/http net/http/httptest
func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/products", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestGetNonExistentProduct(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/product/11", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)
	if m["error"] != "Product not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got '%s'", m["error"])
	}
}

// tom: rewritten function
func TestCreateProduct(t *testing.T) {

	clearTable()

	var jsonStr = []byte(`{"name":"test product", "price": 11.22}`)
	req, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	response := executeRequest(req)
	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["name"] != "test product" {
		t.Errorf("Expected product name to be 'test product'. Got '%v'", m["name"])
	}

	if m["price"] != 11.22 {
		t.Errorf("Expected product price to be '11.22'. Got '%v'", m["price"])
	}

	// the id is compared to 1.0 because JSON unmarshaling converts numbers to
	// floats, when the target is a map[string]interface{}
	if m["id"] != 1.0 {
		t.Errorf("Expected product ID to be '1'. Got '%v'", m["id"])
	}
}

func TestGetProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)
}

func addProducts(count int) {
	if count < 1 {
		count = 1
	}

	for i := 0; i < count; i++ {
		a.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "Product "+strconv.Itoa(i), (i+1.0)*10)
	}
}

func TestUpdateProduct(t *testing.T) {

	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(req)
	var originalProduct map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &originalProduct)

	var jsonStr = []byte(`{"name":"test product - updated name", "price": 11.22}`)
	req, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	// req, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(payload))
	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["id"] != originalProduct["id"] {
		t.Errorf("Expected the id to remain the same (%v). Got %v", originalProduct["id"], m["id"])
	}

	if m["name"] == originalProduct["name"] {
		t.Errorf("Expected the name to change from '%v' to '%v'. Got '%v'", originalProduct["name"], m["name"], m["name"])
	}

	if m["price"] == originalProduct["price"] {
		fmt.Printf(" %T, %T, %T ", originalProduct["price"], m["price"], m["price"])
		t.Errorf("Expected the price to change from '%v' to '%v'. Got '%v'", originalProduct["price"], m["price"], m["price"])
	}

}

func TestDeleteProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("DELETE", "/product/1", nil)
	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("GET", "/product/1", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)
}

//func TestAddProductToStore(t *testing.T) {
//	clearTable()
//	addStores(1)
//	var jsonStr = []byte(`[ {"product_id":1,"is_available":true},{"product_id":2,"is_available":true]`)
//
//	req, _ := http.NewRequest("POST", "/store/1", bytes.NewBuffer(jsonStr))
//	req.Header.Set("Content-Type", "application/json")
//
//	rr := executeRequest(req)
//	checkResponseCode(t, http.StatusCreated, rr.Code)
//
//	var m map[string]interface{}
//	json.Unmarshal(rr.Body.Bytes(), &m)
//
//	if m["product_id"] != "1" {
//		t.Errorf("Expected product name to be '%v'. Got '%v'", 1, m["product_id"])
//	}
//
//	if m["is_available"] != true {
//		t.Errorf("Expected product price to be '%v'. Got '%v'", true, m["is_available"])
//	}
//}
//
//func TestAddProductToStore_ProductAvailable(t *testing.T) {
//	clearTable()
//	addProducts(1)
//	var jsonStr = []byte(`[ {"product_id":1,"is_available":true},{"product_id":2,"is_available":true]`)
//
//	req, _ := http.NewRequest("POST", "/store/1", bytes.NewBuffer(jsonStr))
//	req.Header.Set("Content-Type", "application/json")
//
//	rr := executeRequest(req)
//	checkResponseCode(t, http.StatusCreated, rr.Code)
//
//	var m map[string]interface{}
//	json.Unmarshal(rr.Body.Bytes(), &m)
//
//	if m["product_id"] != "1" {
//		t.Errorf("Expected product name to be '%v'. Got '%v'", 1, m["product_id"])
//	}
//
//	if m["is_available"] != true {
//		t.Errorf("Expected product price to be '%v'. Got '%v'", true, m["is_available"])
//	}
//}
//
//func addStores(count int) {
//	if count < 1 {
//		count = 1
//	}
//
//	for i := 0; i < count; i++ {
//		a.GormDB.Exec("INSERT INTO stores(store_id, name) VALUES($1, $2)", 1, i+1)
//	}
//}
//
//func TestGetProductsInStore(t *testing.T) {
//	clearTable()
//	addProducts(1)
//	addStores(1)
//
//	req, _ := http.NewRequest("GET", "/store/1/products", nil)
//	rr := executeRequest(req)
//	checkResponseCode(t, http.StatusOK, rr.Code)
//}
