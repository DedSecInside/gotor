const HtmlWebPackPlugin = require("html-webpack-plugin");

const htmlPlugin = new HtmlWebPackPlugin({
  template: "./src/index.html",
  filename: "./index.html"
});


module.exports = {
  entry: {
    main: "./src/index.jsx"
  },
  resolve: {
    extensions: ['*', '.js', '.jsx']
  },
  module: {
    rules: [{
      test: /\.(js|jsx)?$/,
      exclude: /node_modules/,
      use: ['babel-loader']
    }, {
      test: /\.css$/,
      loader: "style-loader!css-loader"
    }]
  },
  plugins: [htmlPlugin]
};
