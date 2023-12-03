package shawn.rego

default ruleAny = false
default ruleAdminOnly = false
default ruleUserOnly = false
default ruleAdminOrSubject = false

roleUser := "USER"
roleAdmin := "ADMIN"
roleAll := {roleAdmin, roleUser}


# "role |": This introduces the set comprehension. It's saying, "for each element that meets the following criteria, add it to the set."
# what is set in rego? {"USER", "ADMIN"} is set
# what is set comprehension? {role | role := input.Roles[_]} is set comprehension
# put everything together: if input.Roles contained ["ADMIN", "USER"], then the set comprehension would produce the set {"ADMIN", "USER"}.
ruleAny {
	claim_roles := {role | role := input.Roles[_]}
	input_roles := roleAll & claim_roles
	count(input_roles) > 0
}

# becuase we are comparing set, need to use {roleAdmin} with {}
ruleAdminOnly {
	claim_roles := {role | role := input.Roles[_]}
	input_admin := {roleAdmin} & claim_roles
	count(input_admin) > 0
}

# sample input to evaluate ruleAdminOnly:
# {
#     "UserID": "12345",
#     "Subject": "12345",
#     "Roles": ["USER"]
# }
ruleUserOnly {
	claim_roles := {role | role := input.Roles[_]}
	input_user := {roleUser} & claim_roles
	count(input_user) > 0
}

ruleAdminOrSubject {
	claim_roles := {role | role := input.Roles[_]}
	input_admin := {roleAdmin} & claim_roles
    count(input_admin) > 0
} else {
    claim_roles := {role | role := input.Roles[_]}
	input_user := {roleUser} & claim_roles
	count(input_user) > 0
	input.UserID == input.Subject
}
