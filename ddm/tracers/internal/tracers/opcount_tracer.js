
{

	count: 0,

	step: function(log, db) { this.count++ },

	fault: function(log, db) { },

	result: function(ctx, db) { return this.count }
}
