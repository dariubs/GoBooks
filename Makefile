.PHONY: validate
validate:
	cd validator && go run . ../README.md
