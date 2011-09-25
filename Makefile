# Modified the basic makefiles referred to from the
# Go home page.
#
# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

TARG=goline
GOFILES=\
		strings.go\
		errors.go\
		set.go\
		question.go\
		menu.go\
        goline.go\

include $(GOROOT)/src/Make.pkg

ex: install force
	bash -c 'for d in examples/*; do cd $$d && gomake && cd -; done'

exinstall: force
	bash -c 'for d in examples/*; do cd $$d && gomake install && cd -; done'

exclean: force
	bash -c 'for d in examples/*; do cd $$d && gomake clean && cd -; done'

exnuke: force
	bash -c 'for d in examples/*; do cd $$d && gomake nuke && cd -; done'

force:
