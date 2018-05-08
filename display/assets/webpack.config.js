
const webpack = require('webpack');
const path = require('path');

module.exports = {
	resolve: {
		extensions: ['.js', '.jsx'],
	},
	entry:  './index',
	output: {
		path:     path.resolve(__dirname, ''),
		filename: 'bundle.js',
	},
	plugins: [
		new webpack.optimize.UglifyJsPlugin({
			comments: false,
			mangle:   false,
			beautify: true,
		}),
	],
	module: {
		rules: [
			{
				test:    /\.jsx$/, 
				exclude: /node_modules/,
				use:     [ 
					{
						loader:  'babel-loader',
						options: {
							plugins: [ 

								'transform-class-properties', 
								'transform-flow-strip-types',
							],
							presets: [ 
								'env',
								'react',
								'stage-0',
							],
						},
					},

				],
			},
			{
				test: /font-awesome\.css$/,
				use:  [
					'style-loader',
					'css-loader',
					path.resolve(__dirname, './fa-only-woff-loader.js'),
				],
			},
			{
				test: /\.woff2?$/, 
				use:  'url-loader',
			},
		],
	},
};
