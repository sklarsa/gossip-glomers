maelstrom_version ?= v0.2.2

.PHONY: install-mac
install-mac:
	echo $(maelstrom_version)
	brew install openjdk graphviz gnuplot
	gh release download $(maelstrom_version) -R jepsen-io/maelstrom -O - | tar -xvf -

.PHONY: run-1-echo
run-1-echo:
	cd challenges/maelstrom-1-echo && go install .
	cd maelstrom && ./maelstrom test -w echo --bin ~/go/bin/maelstrom-1-echo --node-count 1 --time-limit 10

.PHONY: run-2-uid-generation
run-2-uid-generation:
	cd challenges/maelstrom-2-uid-generation && go install .
	cd maelstrom && ./maelstrom test -w unique-ids --bin ~/go/bin/maelstrom-2-uid-generation --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

.PHONY: run-3-broadcast
run-3-broadcast:
	cd challenges/maelstrom-3-broadcast && go install .
	cd maelstrom && ./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-3-broadcast --node-count 1 --time-limit 20 --rate 10

.PHONY: run-4-g-counter
run-4-g-counter:
	cd challenges/maelstrom-4-g-counter && go install .
	cd maelstrom && ./maelstrom test -w g-counter --bin ~/go/bin/maelstrom-4-g-counter --node-count 3 --rate 50 --time-limit 5