maelstrom_version ?= v0.2.2

.PHONY: install-mac
install-mac:
	echo $(maelstrom_version)
	brew install openjdk graphviz gnuplot gh
	gh release download $(maelstrom_version) -R jepsen-io/maelstrom -O - | tar -xvf -

.PHONY: run-1-echo
run-1-echo:
	cd challenges/maelstrom-1-echo && go install .
	cd maelstrom && ./maelstrom test -w echo --bin ~/go/bin/maelstrom-1-echo --node-count 1 --time-limit 10

.PHONY: run-2-uid-generation
run-2-uid-generation:
	cd challenges/maelstrom-2-uid-generation && go install .
	cd maelstrom && ./maelstrom test -w unique-ids --bin ~/go/bin/maelstrom-2-uid-generation --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

.PHONY: run-3a-broadcast
run-3a-broadcast:
	cd challenges/maelstrom-3a-broadcast && go install .
	cd maelstrom && ./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-3a-broadcast --node-count 1 --time-limit 20 --rate 10

.PHONY: run-3b-broadcast
run-3b-broadcast:
	cd challenges/maelstrom-3b-broadcast && go install .
	cd maelstrom && ./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-3b-broadcast --node-count 5 --time-limit 20 --rate 10

.PHONY: run-3c-broadcast
run-3c-broadcast:
	cd challenges/maelstrom-3c-broadcast && go install .
	cd maelstrom && ./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-3c-broadcast --node-count 5 --time-limit 5 --rate 10 --nemesis partition

.PHONY: run-4-g-counter
run-4-g-counter:
	cd challenges/maelstrom-4-g-counter && go install .
	cd maelstrom && ./maelstrom test -w g-counter --bin ~/go/bin/maelstrom-4-g-counter --node-count 3 --rate 100 --time-limit 20 --nemesis partition

.PHONY: run-5a-kafka
run-5a-kafka:
	cd challenges/maelstrom-5a-kafka && go install .
	cd maelstrom && ./maelstrom test -w kafka --bin ~/go/bin/maelstrom-5a-kafka --node-count 1 --concurrency 2n --time-limit 20 --rate 1000

.PHONY: run-5b-kafka
run-5b-kafka:
	cd challenges/maelstrom-5b-kafka && go install .
	cd maelstrom && ./maelstrom test -w kafka --bin ~/go/bin/maelstrom-5b-kafka --node-count 2 --concurrency 2n --time-limit 5 --rate 100
