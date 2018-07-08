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
            return <option name='argument' key={arg} value={arg}>{arg}</option>;
          })
        }
      </select>
    )
  }
}


class DisplayURLs extends React.Component {
  constructor(props) {
    super(props);
  }

  goBack() {
    main()
  }

  render() {
    return (
      <table>
        <thead>
          <tr>
            <th> URLS </th>
          </tr>
        </thead>
        <tfoot>
          <tr>
            <td>
              <button onClick={this.goBack} id="backButton">BACK</button>
            </td>
          </tr>
        </tfoot>
        <tbody>
        {
          Object.keys(this.props.websites).map((website, idx) => {
            if (this.props.websites[website] == true) {
              return <tr name="website" key={website}>
                      <td id="goodLink">{idx+1}. {website}</td>
                    </tr>;
            } else {
              return <tr name="website" key={website}>
                      <td id="badLink">{idx+1}. {website}</td>
                    </tr>;
            }
          })
        }
      </tbody>
    </table>
    )
  }
}

class TorBotForm extends React.Component {
  constructor(props) {
    super(props);
    this.state = {option: 'Retrieve Mail', website: ''};
    this.handleChange = this.handleChange.bind(this);
    this.inputChange = this.inputChange.bind(this);
    this.handleSubmit = this.handleSubmit.bind(this);
  }

  inputChange(event) {
    this.setState({option: this.state.option, website: event.target.value});
  }
  handleChange(event) {
    this.setState({option: event.target.value, website: this.state.website});
  }

  handleSubmit(event) {
    event.preventDefault();
    event.stopPropagation();
    if (this.state.option === 'Retrieve URLs') {
      fetch('http://localhost:8008/LIVE', {
        body: JSON.stringify(this.state),
        method: 'POST'
      }).then(response => {
        // object has 'websites' property that contains an array of links
        return response.json();
      }).then(data => {
        debugger;
        handleURLs(data);
      }).catch(error => {
        alert(error);
      });
      }
    }

    render() {
        return (
            <form onSubmit={this.handleSubmit} id ="mainForm">
              <label id='siteFieldTitle'> Website:
                <input onChange={this.inputChange} id='siteName' type='text' name='website'/>
              </label>
              <br/>
              <label id='optionTitle'> Option:
                <TorBotArguments handler={this.handleChange}args={this.props.args}/>
            </label>
            <br/>
            <input id='submitBtn' type="submit"/>
           </form>
        );
    }
}

function handleURLs(data) {
  ReactDOM.render(<DisplayURLs websites={data.websites}/>, document.getElementById('root'));
}

function main() {
  var flags = ['Retrieve Emails', 'Retrieve URLs', 'Retrieve Information']
  ReactDOM.render(<TorBotForm args={flags}/>, document.getElementById('root'));
}

main()

