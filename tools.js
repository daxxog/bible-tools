/* tools.js
 * Tools for parsing The Word of God. An in-depth study.
 * (c) 2018 David (daXXog) Volm ><> + + + <><
 * Released under Apache License, Version 2.0:
 * http://www.apache.org/licenses/LICENSE-2.0.html  
 */

/* UMD LOADER: https://github.com/umdjs/umd/blob/master/returnExports.js */
(function (root, factory) {
    if (typeof exports === 'object') {
        // Node. Does not work with strict CommonJS, but
        // only CommonJS-like enviroments that support module.exports,
        // like Node.
        module.exports = factory();
    } else if (typeof define === 'function' && define.amd) {
        // AMD. Register as an anonymous module.
        define(factory);
    } else {
        // Browser globals (root is window)
        root.BibleTools = factory();
  }
}(this, function() {
	var fs = require('fs'),
		path = require('path'),
		XeTeX = require('./xetex.js')

    var BibleTools = function() {
    	this.sourceDir = './eng-web_xetex';
    	this.books = {};

    	this.parseBook('1CH');
    };

   	BibleTools.prototype.parseBook = function(prefix) {
   		filename = path.join(this.sourceDir, prefix + '_src.tex');

   		var that = this;

   		fs.readFile(filename, 'utf8', function(err, data) {
   			if(!err) {
   				that.books[filename] = new XeTeX(data, prefix);
   				console.log(that.books[filename]);
   			} else {
   				console.error('Could not read book: ' + filename);
   				console.error('Please try running ./get-bible.sh to download a copy of The World English Bible.');
   			}
   		});
   	};

   	new BibleTools();

    return BibleTools;
}));
