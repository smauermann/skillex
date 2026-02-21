
build:
  go build -o skillex

run: build
  ./skillex

vhs: build
  vhs demo.tape
