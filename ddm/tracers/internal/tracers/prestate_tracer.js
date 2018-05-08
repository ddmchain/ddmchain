
{

	prestate: null,

	lookupAccount: function(addr, db){
		var acc = toHex(addr);
		if (this.prestate[acc] === undefined) {
			this.prestate[acc] = {
				balance: '0x' + db.getBalance(addr).toString(16),
				nonce:   db.getNonce(addr),
				code:    toHex(db.getCode(addr)),
				storage: {}
			};
		}
	},

	lookupStorage: function(addr, key, db){
		var acc = toHex(addr);
		var idx = toHex(key);

		if (this.prestate[acc].storage[idx] === undefined) {
			var val = toHex(db.getState(addr, key));
			if (val != "0x0000000000000000000000000000000000000000000000000000000000000000") {
				this.prestate[acc].storage[idx] = toHex(db.getState(addr, key));
			}
		}
	},

	result: function(ctx, db) {

		this.lookupAccount(ctx.from, db);

		var fromBal = bigInt(this.prestate[toHex(ctx.from)].balance.slice(2), 16);
		var toBal   = bigInt(this.prestate[toHex(ctx.to)].balance.slice(2), 16);

		this.prestate[toHex(ctx.to)].balance   = '0x'+toBal.subtract(ctx.value).toString(16);
		this.prestate[toHex(ctx.from)].balance = '0x'+fromBal.add(ctx.value).toString(16);

		this.prestate[toHex(ctx.from)].nonce--;
		if (ctx.type == 'CREATE') {

			delete this.prestate[toHex(ctx.to)];
		}

		return this.prestate;
	},

	step: function(log, db) {

		if (this.prestate === null){
			this.prestate = {};

			this.lookupAccount(log.contract.getAddress(), db);
		}

		switch (log.op.toString()) {
			case "EXTCODECOPY": case "EXTCODESIZE": case "BALANCE":
				this.lookupAccount(toAddress(log.stack.peek(0).toString(16)), db);
				break;
			case "CREATE":
				var from = log.contract.getAddress();
				this.lookupAccount(toContract(from, db.getNonce(from)), db);
				break;
			case "CALL": case "CALLCODE": case "DELEGATECALL": case "STATICCALL":
				this.lookupAccount(toAddress(log.stack.peek(1).toString(16)), db);
				break;
			case 'SSTORE':case 'SLOAD':
				this.lookupStorage(log.contract.getAddress(), toWord(log.stack.peek(0).toString(16)), db);
				break;
		}
	},

	fault: function(log, db) {}
}
