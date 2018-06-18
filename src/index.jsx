import css from './index.css';
import React from "react";
import ReactDOM from "react-dom";

class TorBotArguments extends React.Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <select onChange={this.props.handler}>
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
    console.log(this.state.option);
  }

    render() {
        return (
            <form onSubmit={this.handleSubmit} id ="Main">
              <label> Website:
                <input type="text" name='website'/>
              </label>
              <br/>
              <label> Option:
                <TorBotArguments handler={this.handleChange} args={this.props.args}/>
            </label>
            <br/>
            <input type="submit"/>
           </form>
        );
    }
}

var flags = ['MAIL', 'LIVE', 'INFO']
ReactDOM.render(<TorBotForm args={flags}/>, document.getElementById('root'));


