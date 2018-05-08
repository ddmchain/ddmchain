
module.exports = function(content) {
	return content
		.replace(/src.*url(?!.*url.*(\.eot)).*(\.eot)[^;]*;/,'')
		.replace(/url(?!.*url.*(\.eot)).*(\.eot)[^,]*,/,'')
		.replace(/url(?!.*url.*(\.ttf)).*(\.ttf)[^,]*,/,'')
		.replace(/,[^,]*url(?!.*url.*(\.svg)).*(\.svg)[^;]*;/,';');
};
