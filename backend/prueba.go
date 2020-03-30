package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/buaazp/fasthttprouter"
	"github.com/gocolly/colly"
	_ "github.com/lib/pq"
	"github.com/valyala/fasthttp"
	"github.com/xellio/whois"
)

type domain struct {
	domainID         int
	Name             string   `json:"host"`
	Servers          []server `json:"endpoints"`
	ServerChanged    bool
	SslGrade         string
	PreviousSslGrade string
	Logo             string
	Title            string
	IsDown           bool
}

type server struct {
	Address  string `json:"ipAddress"`
	SslGrade string `json:"grade"`
	Country  string
	Owner    string
}

type input struct {
	In string `json:"name"`
}

type output struct {
	Out string
}

var db *sql.DB

func init() {
	var err error
	connStr := "postgres://root@localhost:26257/defaultdb?sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("Connected to the database")
}

func clasificarGrado(grado string) int {

	switch grado {
	case "A+":
		return 2
	case "A":
		return 3
	case "A-":
		return 4
	case "B+":
		return 5
	case "B":
		return 6
	case "B-":
		return 7
	case "C+":
		return 8
	case "C":
		return 9
	case "C-":
		return 10
	case "D+":
		return 11
	case "D":
		return 12
	case "D-":
		return 13
	case "E+":
		return 14
	case "E":
		return 15
	case "E-":
		return 16
	case "F+":
		return 17
	case "F":
		return 18
	case "F-":
		return 19
	default:
		return 1
	}
}

func retornarGrado(servers []server) string {

	respuesta := ""
	if len(servers) == 1 {
		respuesta = servers[0].SslGrade
	} else if len(servers) > 1 {
		respuesta = servers[0].SslGrade

		for i := 1; i < len(servers); i++ {
			valorRespuesta := clasificarGrado(respuesta)
			valorComparar := clasificarGrado(servers[i].SslGrade)

			if valorRespuesta < valorComparar {
				respuesta = servers[i].SslGrade
			}
		}

	}

	return respuesta

}

func insertDomain(Objdomain *domain) int {

	var err error
	numeroID := 0
	url := "https://www."
	Objdomain.ServerChanged = true
	Objdomain.SslGrade = retornarGrado(Objdomain.Servers)
	Objdomain.PreviousSslGrade = ""
	Objdomain.Logo = obtenerLogo(url + Objdomain.Name)
	Objdomain.Title = obtenertitulo(url + Objdomain.Name)
	Objdomain.IsDown = false

	row, err := db.Query("INSERT INTO tbldomain (domain,server_changed,ssl_grade,previous_ssl_grade,logo,title,is_down) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING domain_id;", Objdomain.Name, Objdomain.ServerChanged, Objdomain.SslGrade, Objdomain.PreviousSslGrade, Objdomain.Logo, Objdomain.Title, Objdomain.IsDown)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}

	for row.Next() {
		err := row.Scan(&numeroID)
		if err != nil {
			fmt.Println(err.Error())
			return 0
		}
	}

	return numeroID

}

//tengo que pasarle el objeto de la base y la nueva consulta
func updateDomain(Objdomain *domain, Basedomain *domain) {

	var err error
	url := "https://www."

	Objdomain.PreviousSslGrade = Basedomain.SslGrade
	Objdomain.SslGrade = retornarGrado(Objdomain.Servers)
	ChangeGrade := (Objdomain.PreviousSslGrade != Objdomain.SslGrade)
	Objdomain.Logo = obtenerLogo(url + Objdomain.Name)
	ChangeLogo := (Objdomain.Logo != Basedomain.Logo)
	Objdomain.Title = obtenertitulo(url + Objdomain.Name)
	ChangeTitle := (Objdomain.Title != Basedomain.Title)
	//Objdomain.IsDown = strconv.Itoa(len(Objdomain.Servers))
	Objdomain.IsDown = false
	Objdomain.ServerChanged = (ChangeGrade || ChangeLogo || ChangeTitle)

	_, err = db.Query("UPDATE tbldomain SET server_changed = $1, ssl_grade = $2, previous_ssl_grade = $3, logo = $4, title = $5,  is_down = $6 WHERE domain_id = $7 ;", Objdomain.ServerChanged, Objdomain.SslGrade, Objdomain.PreviousSslGrade, Objdomain.Logo, Objdomain.Title, Objdomain.IsDown, Basedomain.domainID)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	return

}

func insertServer(Objdomain *domain, idInsert int) {

	for _, server := range Objdomain.Servers {

		insertOneServer(&server, idInsert)
	}

	return

}

func insertOneServer(Objserver *server, idInsert int) {

	var err error

	Objserver.Owner = obtenerOwner(Objserver.Address)
	Objserver.Country = obtenerCountry(Objserver.Address)

	_, err = db.Query("INSERT INTO tblserver (dominio,address,ssl_grade,country,owner) VALUES ($1,$2,$3,$4,$5);", idInsert, Objserver.Address, Objserver.SslGrade, Objserver.Country, Objserver.Owner)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	return

}

func actualizarServidor(ip string, servers []server) bool {

	for _, server := range servers {

		if ip == server.Address {
			return true
		}
	}

	return false

}

func updateServer(Objdomain *domain, Basedomain domain) {

	idInsert := Basedomain.domainID

	var err error
	for _, server := range Objdomain.Servers {

		actualizar := actualizarServidor(server.Address, Basedomain.Servers)

		if actualizar {
			nuevogrado := server.SslGrade
			_, err = db.Query("UPDATE tblserver SET ssl_grade = $1 WHERE dominio = $2 AND address = $3;", nuevogrado, idInsert, server.Address)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		} else {
			insertOneServer(&server, idInsert)
		}

	}

	return

}

func traerTodoDominio() ([]domain, error) {

	var err error
	Dominios := make([]domain, 0)
	rows, err := db.Query("SELECT * FROM tbldomain;")

	if err != nil {
		fmt.Println(err.Error())
		return Dominios, err
	} // (2)
	defer rows.Close()

	for rows.Next() {
		dominio := domain{}

		err := rows.Scan(&dominio.domainID, &dominio.Name, &dominio.ServerChanged, &dominio.SslGrade, &dominio.PreviousSslGrade, &dominio.Logo, &dominio.Title, &dominio.IsDown)
		if err != nil {
			Dominios := make([]domain, 0)
			fmt.Println(err.Error())
			return Dominios, err
		}

		ser, err := traerTodoServer(dominio.domainID)
		if err != nil {
			Dominios := make([]domain, 0)
			fmt.Println(err.Error())
			return Dominios, err
		}

		dominio.Servers = ser

		Dominios = append(Dominios, dominio)
	}

	if err = rows.Err(); err != nil {
		Dominios := make([]domain, 0)
		fmt.Println(err.Error())
		return Dominios, err
	}

	return Dominios, nil

}

func traerDominio(nombre string, servers []server) (domain, int, error) {

	var err error
	dominio := domain{}
	rows, err := db.Query("SELECT * FROM tbldomain WHERE domain ILIKE $1;", nombre)

	if err != nil {
		fmt.Println(err.Error())
		return dominio, 0, err
	} // (2)
	defer rows.Close()
	count := 0
	for rows.Next() {
		dominio = domain{}
		count++
		err := rows.Scan(&dominio.domainID, &dominio.Name, &dominio.ServerChanged, &dominio.SslGrade, &dominio.PreviousSslGrade, &dominio.Logo, &dominio.Title, &dominio.IsDown)
		if err != nil {

			fmt.Println(err.Error())
			return dominio, 0, err
		}
		ser := make([]server, 0)
		if len(servers) == 0 {
			ser, err = traerTodoServer(dominio.domainID)
		} else {
			ser, err = traerServer(servers, dominio.domainID)
		}

		if err != nil {

			fmt.Println(err.Error())
			return dominio, 0, err
		}

		dominio.Servers = ser
	}

	if err = rows.Err(); err != nil {
		fmt.Println(err.Error())
		return dominio, 0, err
	}

	return dominio, count, nil

}

func traerTodoServer(idDomain int) ([]server, error) {

	var err error
	Servers := make([]server, 0)
	rows, err := db.Query("SELECT address,ssl_grade,country,owner FROM tblserver WHERE dominio = $1;", idDomain)

	if err != nil {
		fmt.Println(err.Error())
		return Servers, err
	} // (2)
	defer rows.Close()

	for rows.Next() {
		server := server{}

		err := rows.Scan(&server.Address, &server.SslGrade, &server.Country, &server.Owner)
		if err != nil {
			fmt.Println(err.Error())
			return Servers, err
		}
		Servers = append(Servers, server)
	}

	if err = rows.Err(); err != nil {
		Servers := make([]server, 0)
		fmt.Println(err.Error())
		return Servers, err
	}

	return Servers, nil

}

func listarServersInt(Servers []server) string {

	Ips := make([]string, 0)
	for _, resto := range Servers {
		ip := resto.Address
		Ips = append(Ips, ip)
	}

	respuesta := "'" + strings.Join(Ips, "','") + "'"

	return respuesta
}

func deleteServer(serversBusqueda []server, idDomain int) {

	var err error
	lista := listarServersInt(serversBusqueda)
	sqlDel := "DELETE FROM tblserver WHERE dominio = '" + strconv.Itoa(idDomain) + "' AND address in (" + lista + ");"

	//rows, err := db.Query("DELETE FROM tblserver WHERE dominio = $1 AND address in ($2);", idDomain, lista)
	rows, err := db.Query(sqlDel)

	if err != nil {
		fmt.Println(err.Error())
		return
	} // (2)
	defer rows.Close()

	for rows.Next() {
		respuesta := "nada"
		err := rows.Scan(respuesta)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		println(respuesta)
	}

	return
}

func traerServer(serversBusqueda []server, idDomain int) ([]server, error) {

	var err error
	Servers := make([]server, 0)
	lista := listarServersInt(serversBusqueda)

	sqlSel := "SELECT address,ssl_grade,country,owner FROM tblserver WHERE dominio = '" + strconv.Itoa(idDomain) + "' AND address in (" + lista + ");"
	//rows, err := db.Query("SELECT address,ssl_grade,country,owner FROM tblserver WHERE dominio = $1 and address in ($2); ", idDomain, pq.Array(lista))
	rows, err := db.Query(sqlSel)
	if err != nil {
		fmt.Println(err.Error())
		return Servers, err
	} // (2)
	defer rows.Close()

	for rows.Next() {
		server := server{}

		err := rows.Scan(&server.Address, &server.SslGrade, &server.Country, &server.Owner)
		if err != nil {
			fmt.Println(err.Error())
			return Servers, err
		}
		Servers = append(Servers, server)
	}

	if err = rows.Err(); err != nil {
		Servers := make([]server, 0)
		fmt.Println(err.Error())
		return Servers, err
	}

	return Servers, nil

}

func consulta(ctx *fasthttp.RequestCtx) {

	var err error
	output := output{"inicio de la salida"}

	_, err = json.Marshal(output)
	if err != nil {
		fmt.Println(err)
	}

	body := ctx.PostBody()
	input := input{}

	err = json.Unmarshal(body, &input)
	if err != nil {
		output.Out = err.Error()
		fmt.Println(err.Error())
		return
	}

	//dominio := input.In

	dominios, err := traerTodoDominio()

	for _, domimio := range dominios {
		_, err := traerTodoServer(domimio.domainID)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

	}

	outjson1, err := json.Marshal(dominios)
	if err != nil {
		fmt.Println(err)
	}

	ctx.Response.Header.Set("Content-Type", "application/json; charset=UTF-8")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(ctx, string(outjson1))

}

func index(ctx *fasthttp.RequestCtx) {

	var err error

	output := output{"inicio de la salida"}

	_, err = json.Marshal(output)
	if err != nil {
		fmt.Println(err)
	}

	body := ctx.PostBody()
	input := input{}

	err = json.Unmarshal(body, &input)
	if err != nil {
		output.Out = err.Error()
		fmt.Println(err.Error())
		return
	}

	dominio := input.In

	enlace := "https://api.ssllabs.com/api/v3/analyze?host="

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(enlace + dominio)

	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	client.Do(req, resp)

	bodyBytes := resp.Body()

	infoDominio := domain{}

	err = json.Unmarshal(bodyBytes, &infoDominio)
	if err != nil {
		output.Out = err.Error()
		fmt.Println(err.Error())
		return
	}

	output.Out = string(bodyBytes)

	//consultar si esta el dominio

	if infoDominio.Name != "" {

		infoDominioBase, num, err := traerDominio(dominio, infoDominio.Servers)

		if err != nil {
			output.Out = err.Error()
			fmt.Println(err.Error())
			return
		}

		if num == 0 {

			idInsert := insertDomain(&infoDominio)

			insertServer(&infoDominio, idInsert)

		} else {
			updateDomain(&infoDominio, &infoDominioBase)
			//deleteServer(infoDominio.Servers, infoDominioBase.domainID)
			//si ya esta y no esta el servidor hay que agregarlo
			updateServer(&infoDominio, infoDominioBase)
		}

		infoDominioBase, _, err = traerDominio(dominio, infoDominio.Servers)

		infoDominio = infoDominioBase
	}

	outjson1, err := json.Marshal(infoDominio)
	if err != nil {
		fmt.Println(err)
	}

	ctx.Response.Header.Set("Content-Type", "application/json; charset=UTF-8")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(ctx, string(outjson1))

}

func obtenertitulo(url string) string {

	c := colly.NewCollector()

	respuesta := ""

	c.OnHTML("title", func(e *colly.HTMLElement) {
		respuesta = e.Text
	})

	c.Visit(url)

	return respuesta

}

func obtenerLogo(url string) string {

	c := colly.NewCollector()

	respuesta := ""

	c.OnHTML("link", func(e *colly.HTMLElement) {
		if e.Attr("rel") == "shortcut icon" {

			respuesta = e.Attr("href")
		}

	})

	c.Visit(url)

	return respuesta

}

func obtenerOwner(numeroIP string) string {

	respuesta := ""

	ip := net.ParseIP(numeroIP)
	res, err := whois.QueryIP(ip)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	if len(res.Output["Registrant Organization"]) >= 1 {
		respuesta = res.Output["Registrant Organization"][0]
	}

	return respuesta

}

func obtenerCountry(numeroIP string) string {

	respuesta := ""

	ip := net.ParseIP(numeroIP)
	res, err := whois.QueryIP(ip)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	if len(res.Output["Registrant Country"]) >= 1 {
		respuesta = res.Output["Registrant Country"][0]
	}

	return respuesta

}

func hello(ctx *fasthttp.RequestCtx) {

	ip := net.ParseIP("52.3.102.88")
	_, err := whois.QueryIP(ip)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {

	router := fasthttprouter.New()
	router.GET("/", hello)
	router.POST("/", index)
	router.POST("/informacion", consulta)

	log.Fatal(fasthttp.ListenAndServe(":8090", router.Handler))
}
