package jwtmethod

// claims := jwt.RegisteredClaims{
// 		Subject:   strconv.FormatInt(resp.UserId, 10),
// 		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
// 		IssuedAt:  jwt.NewNumericDate(time.Now()),
// 		Issuer:    "api-gateway",
// 	}
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	signedToken, err := token.SignedString([]byte(s.jwtSecret))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to sign token: %w", err)
// 	}
// 	return signedToken, nil
