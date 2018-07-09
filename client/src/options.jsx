import React from "react";
import ReactDOM from "react-dom";
import TorBotForm from "./index.jsx";

class DisplayURLs extends React.Component {
  constructor(props) {
    super(props);
  }

  goBack() {
    ReactDOM.render(<TorBotForm args={this.props.flags}/>, document.getElementById('root'))
  }

  render() {
    return (
      <table>
        <thead>
          <tr>
            <th> URLS FOUND </th>
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

export default DisplayURLs;
