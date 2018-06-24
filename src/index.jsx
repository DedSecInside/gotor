import css from './index.css';
import React from "react";
import ReactDOM from "react-dom";

class TorBotArguments extends React.Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <select className="dropdown-select" onChange={this.props.handler}>
        {
        this.props.args.map((arg) => {
            return <option name='argument' value={arg}>{arg}</option>;
          })
        }
      </select>
    )
  }
}

class TorBotForm extends React.Component {
  constructor(props) {
    super(props);
    this.state = {option: 'MAIL'};
    this.handleChange = this.handleChange.bind(this);
    this.handleSubmit = this.handleSubmit.bind(this);
  }

  handleChange(event) {
    this.setState({option: event.target.value});
  }

  handleSubmit(event) {
    if (this.state.option === 'LIVE') {
      fetch('http://localhost:8080/LIVE').then(data => {
        return data.json();
      }).then(data => {
        console.log(data);
        debugger;
      }).catch(error => {
        alert(error);
        debugger;
      });
      }
    }

    render() {
        return (
            <form onSubmit={this.handleSubmit} id ="mainForm">
              <label id='siteFieldTitle'> Website:
                <input id='siteName' type='text' name='website'/>
              </label>
              <br/>
              <label id='optionTitle'> Option:
                <TorBotArguments handler={this.handleChange} args={this.props.args}/>
            </label>
            <br/>
            <input id='submitBtn' type="submit"/>
           </form>
        );
    }
}

var flags = ['MAIL', 'LIVE', 'INFO']
ReactDOM.render(<TorBotForm args={flags}/>, document.getElementById('root'));


