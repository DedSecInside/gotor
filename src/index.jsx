import React from "react";
import ReactDOM from "react-dom";

class TorBotArguments extends React.Component {
  render() {
    return (
      <select>
        {
          this.props.args.map(function(arg) {
            return <option value={arg}>{arg}</option>;
          })
        }
      </select>
    )
  }
}

var activateTorBot = (event) => {
  alert('Button Submitted');
};

class TorBotForm extends React.Component {

    render() {
        return (
            <form onSubmit={activateTorBot} id ="Main">
              <label> Website:
                <input type="text" name='website'/>
              </label>
              <br/>
              <label> Option:
                <TorBotArguments args={this.props.args}/>
            </label>
            <br/>
            <input type="submit"/>
           </form>
        );
    }
}

var flags = ['mail', 'live', 'info']
ReactDOM.render(<TorBotForm args={flags}/>, document.getElementById('root'));


