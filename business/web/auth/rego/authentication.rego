package shawn.rego

default auth = false

# auth is allowed if jwt_valid evaluates to true
auth {
	jwt_valid
}

# The verify_jwt function returns three variables: 
# valid (a boolean indicating if the JWT is valid), 
# header (the JWT header), 
# payload (the JWT payload).
jwt_valid := valid {
	[valid, header, payload] := verify_jwt
}

verify_jwt := io.jwt.decode_verify(input.Token, {
        "cert": input.Key,
        "iss": input.ISS,
	}
)
