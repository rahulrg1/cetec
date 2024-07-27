package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/go-sql-driver/mysql"
)

type PersonInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

var DB *sql.DB

func main() {
	connectionString := "root:password@tcp(127.0.0.1:3306)/cetec"

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to the cetec DB!")

	router := gin.Default()

	router.GET("/person/:person_id/info", func(c *gin.Context) {
		personID := c.Param("person_id")

		personQuery := `
            SELECT p.name, ph.number AS phone_number, a.city, a.state, a.street1, a.street2, a.zip_code
            FROM person p
            JOIN phone ph ON p.id = ph.person_id
            JOIN address_join aj ON p.id = aj.person_id
            JOIN address a ON aj.address_id = a.id
            WHERE p.id = ?`

		row := db.QueryRow(personQuery, personID)

		var info PersonInfo
		err := row.Scan(&info.Name, &info.PhoneNumber, &info.City, &info.State, &info.Street1, &info.Street2, &info.ZipCode)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"message": "Person not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, info)
	})

	router.POST("/person/create", func(c *gin.Context) {
		var req PersonInfo
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
			return
		}
		defer tx.Rollback()

		personResult, err := tx.Exec(`INSERT INTO person (name) VALUES (?)`, req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert person"})
			return
		}
		personID, err := personResult.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get person ID"})
			return
		}

		_, err = tx.Exec(`INSERT INTO phone (person_id, number) VALUES (?, ?)`, personID, req.PhoneNumber)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert phone"})
			return
		}

		addressResult, err := tx.Exec(`INSERT INTO address (city, state, street1, street2, zip_code) VALUES (?, ?, ?, ?, ?)`,
			req.City, req.State, req.Street1, req.Street2, req.ZipCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert address"})
			return
		}
		addressID, err := addressResult.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get address ID"})
			return
		}

		_, err = tx.Exec(`INSERT INTO address_join (person_id, address_id) VALUES (?, ?)`, personID, addressID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert address join"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Person created successfully"})
	})

	router.Run(":8080")
}
