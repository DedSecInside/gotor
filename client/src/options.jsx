import React from 'react';
import ReactDOM from 'react-dom';
import {TorBotForm} from './index';

const DisplayURLs = (props) => {

    const goBack = () => {
        ReactDOM.render(<TorBotForm args={this.props.flags}/>, document.getElementById('root'))
    };

    return (
        <table>
            <thead>
            <tr>
                <th> URLS FOUND</th>
            </tr>
            </thead>
            <tfoot>
            <tr>
                <td>
                    <button onClick={goBack} id="backButton">BACK</button>
                </td>
            </tr>
            </tfoot>
            <tbody>
            {
                Object.keys(props.websites).map((website, idx) =>
                    <tr key={website}>
                        <td className={props.websites[website] ? 'goodLink' : 'badLink'}>{idx + 1}. {website}</td>
                    </tr>
                )
            }
            </tbody>
        </table>
    )
};

export default DisplayURLs;
