/* xetex.js
 * simple parser for bible tex files
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
        root.XeTeX = factory();
  }
}(this, function() {
	var S = require('string');

    var XeTeX = function(texdata, short) {
    	this.short = short;

    	this.parse(texdata);
    };

    XeTeX.prototype.parse = function(texdata) {
    	var line = S(texdata).lines().map(function(line) {
    		return S(line);
    	}), ofs = 0;

    	if((line[0].count('{') === 1) && (line[0].count('}') === 1)) {
			this.name = line[0].between('{', '}').s;

			if(line[1].startsWith('{\\MT ')) {
				this.description = line[1].chompLeft('{\\MT ').s;

				if(line[2].length === 0) {
					//meat and potatoes
				} else {
					this.parseError(line, 2);
				}
			} else {
				this.parseError(line, 1);
			}
		} else {
			this.parseError(line, 0);
		}

    	return line[0].s;
    };

    XeTeX.prototype.parseError = function(line, num) {
    	console.error('Parse error! @ line #' + (num + 1) + ' : ' + line[num]);
    };

    return XeTeX;
}));
