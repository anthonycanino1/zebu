all: spec
	
spec:	spec.dvi
	dvipdf spec.dvi

spec.dvi: spec.tex
	latex spec.tex

clean:
	rm *.log *.out *.dvi *.pdf
