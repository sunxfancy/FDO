PWD=$(shell pwd)
LLVM=$(PWD)/../LLVM-IPRA/llvm
FDO=$(PWD)/../install/FDO

exmaple:
	mkdir -p build-example
	cd build-example && \
		$(FDO) config ../demo && \
		$(FDO) build --lto=thin --pgo --propeller && \
		$(FDO) test --pgo --propeller && \
		$(FDO) opt --pgo --propeller && \
		$(FDO) test --pgo-and-propeller && \
		$(FDO) opt --pgo-and-propeller

run:
	make clang
	cd build-clang && $(FDO) build --lto=thin -s=../clang/FDO_test.yaml -i --pgo --propeller
	make instrumented.bench
	make labeled.bench
	cd build-clang && $(FDO) test  --pgo --propeller
	cd build-clang && $(FDO) opt  --pgo --propeller
	cd build-clang && $(FDO) build --lto=thin -s=../clang/FDO_test.yaml -i --pgo-and-propeller
	make labeled-pgo.bench
	cd build-clang && $(FDO) test --pgo-and-propeller
	cd build-clang && $(FDO) opt --pgo-and-propeller

clang:
	mkdir -p build-clang
	cd build-clang && $(FDO) config $(LLVM) \
		-G Ninja \
		-DCMAKE_BUILD_TYPE=Release \
		-DLLVM_TARGETS_TO_BUILD=X86 \
		-DLLVM_OPTIMIZED_TABLEGEN=On \
		-DLLVM_ENABLE_PROJECTS="clang" 
	
%.bench:
	mkdir -p clang/test/$(basename $@)
	cd clang/test/$(basename $@) && cmake -G Ninja $(LLVM) \
		-DCMAKE_BUILD_TYPE=Release \
		-DLLVM_TARGETS_TO_BUILD=X86 \
		-DLLVM_OPTIMIZED_TABLEGEN=On \
		-DCMAKE_C_COMPILER=$(PWD)/build-clang/$(basename $@)/install/bin/clang \
		-DCMAKE_CXX_COMPILER=$(PWD)/build-clang/$(basename $@)/install/bin/clang++ \
		-DLLVM_ENABLE_PROJECTS="clang" 
	cd clang/test/$(basename $@) && (ninja -t commands | head -100 > $(PWD)/build-clang/$(basename $@)/perf_commands.sh)
	cd $(PWD)/build-clang/$(basename $@) && chmod +x ./perf_commands.sh

.PHONY: clang run