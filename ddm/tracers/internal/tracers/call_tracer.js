
{

	callstack: [{}],

	descended: false,

	step: function(log, db) {

		var error = log.getError();
		if (error !== undefined) {
			this.fault(log, db);
			return;
		}

		var syscall = (log.op.toNumber() & 0xf0) == 0xf0;
		if (syscall) {
			var op = log.op.toString();
		}

		if (syscall && op == 'CREATE') {
			var inOff = log.stack.peek(1).valueOf();
			var inEnd = inOff + log.stack.peek(2).valueOf();

			var call = {
				type:    op,
				from:    toHex(log.contract.getAddress()),
				input:   toHex(log.memory.slice(inOff, inEnd)),
				gasIn:   log.getGas(),
				gasCost: log.getCost(),
				value:   '0x' + log.stack.peek(0).toString(16)
			};
			this.callstack.push(call);
			this.descended = true
			return;
		}

		if (syscall && op == 'SELFDESTRUCT') {
			var left = this.callstack.length;
			if (this.callstack[left-1].calls === undefined) {
				this.callstack[left-1].calls = [];
			}
			this.callstack[left-1].calls.push({type: op});
			return
		}

		if (syscall && (op == 'CALL' || op == 'CALLCODE' || op == 'DELEGATECALL' || op == 'STATICCALL')) {

			var to = toAddress(log.stack.peek(1).toString(16));
			if (isPrecompiled(to)) {
				return
			}
			var off = (op == 'DELEGATECALL' || op == 'STATICCALL' ? 0 : 1);

			var inOff = log.stack.peek(2 + off).valueOf();
			var inEnd = inOff + log.stack.peek(3 + off).valueOf();

			var call = {
				type:    op,
				from:    toHex(log.contract.getAddress()),
				to:      toHex(to),
				input:   toHex(log.memory.slice(inOff, inEnd)),
				gasIn:   log.getGas(),
				gasCost: log.getCost(),
				outOff:  log.stack.peek(4 + off).valueOf(),
				outLen:  log.stack.peek(5 + off).valueOf()
			};
			if (op != 'DELEGATECALL' && op != 'STATICCALL') {
				call.value = '0x' + log.stack.peek(2).toString(16);
			}
			this.callstack.push(call);
			this.descended = true
			return;
		}

		if (this.descended) {
			if (log.getDepth() >= this.callstack.length) {
				this.callstack[this.callstack.length - 1].gas = log.getGas();
			} else {

			}
			this.descended = false;
		}

		if (syscall && op == 'REVERT') {
			this.callstack[this.callstack.length - 1].error = "execution reverted";
			return;
		}
		if (log.getDepth() == this.callstack.length - 1) {

			var call = this.callstack.pop();

			if (call.type == 'CREATE') {

				call.gasUsed = '0x' + bigInt(call.gasIn - call.gasCost - log.getGas()).toString(16);
				delete call.gasIn; delete call.gasCost;

				var ret = log.stack.peek(0);
				if (!ret.equals(0)) {
					call.to     = toHex(toAddress(ret.toString(16)));
					call.output = toHex(db.getCode(toAddress(ret.toString(16))));
				} else if (call.error === undefined) {
					call.error = "internal failure"; 
				}
			} else {

				if (call.gas !== undefined) {
					call.gasUsed = '0x' + bigInt(call.gasIn - call.gasCost + call.gas - log.getGas()).toString(16);

					var ret = log.stack.peek(0);
					if (!ret.equals(0)) {
						call.output = toHex(log.memory.slice(call.outOff, call.outOff + call.outLen));
					} else if (call.error === undefined) {
						call.error = "internal failure"; 
					}
				}
				delete call.gasIn; delete call.gasCost;
				delete call.outOff; delete call.outLen;
			}
			if (call.gas !== undefined) {
				call.gas = '0x' + bigInt(call.gas).toString(16);
			}

			var left = this.callstack.length;
			if (this.callstack[left-1].calls === undefined) {
				this.callstack[left-1].calls = [];
			}
			this.callstack[left-1].calls.push(call);
		}
	},

	fault: function(log, db) {

		if (this.callstack[this.callstack.length - 1].error !== undefined) {
			return;
		}

		var call = this.callstack.pop();
		call.error = log.getError();

		if (call.gas !== undefined) {
			call.gas = '0x' + bigInt(call.gas).toString(16);
			call.gasUsed = call.gas
		}
		delete call.gasIn; delete call.gasCost;
		delete call.outOff; delete call.outLen;

		var left = this.callstack.length;
		if (left > 0) {
			if (this.callstack[left-1].calls === undefined) {
				this.callstack[left-1].calls = [];
			}
			this.callstack[left-1].calls.push(call);
			return;
		}

		this.callstack.push(call);
	},

	result: function(ctx, db) {
		var result = {
			type:    ctx.type,
			from:    toHex(ctx.from),
			to:      toHex(ctx.to),
			value:   '0x' + ctx.value.toString(16),
			gas:     '0x' + bigInt(ctx.gas).toString(16),
			gasUsed: '0x' + bigInt(ctx.gasUsed).toString(16),
			input:   toHex(ctx.input),
			output:  toHex(ctx.output),
			time:    ctx.time,
		};
		if (this.callstack[0].calls !== undefined) {
			result.calls = this.callstack[0].calls;
		}
		if (this.callstack[0].error !== undefined) {
			result.error = this.callstack[0].error;
		} else if (ctx.error !== undefined) {
			result.error = ctx.error;
		}
		if (result.error !== undefined) {
			delete result.output;
		}
		return this.finalize(result);
	},

	finalize: function(call) {
		var sorted = {
			type:    call.type,
			from:    call.from,
			to:      call.to,
			value:   call.value,
			gas:     call.gas,
			gasUsed: call.gasUsed,
			input:   call.input,
			output:  call.output,
			error:   call.error,
			time:    call.time,
			calls:   call.calls,
		}
		for (var key in sorted) {
			if (sorted[key] === undefined) {
				delete sorted[key];
			}
		}
		if (sorted.calls !== undefined) {
			for (var i=0; i<sorted.calls.length; i++) {
				sorted.calls[i] = this.finalize(sorted.calls[i]);
			}
		}
		return sorted;
	}
}
