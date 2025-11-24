package main

import (
	"fmt"
	"os"

	"anomaly-detector/internal/auth"
)

func main() {
	// Point to the mounted PEM in the runtime container
	os.Setenv("KEY_JWT_PRIVATE_KEY_FILE", "/etc/anomaly-detector/keys/jwt_private.pem")
	jwks, err := auth.GetJwks()
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}
	fmt.Printf("JWKS keys count: %d\n", len(jwks["keys"].([]interface{})))
}
