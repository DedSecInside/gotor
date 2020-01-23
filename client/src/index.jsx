import React, {useState} from "react";
import ReactDOM from "react-dom";
import DisplayURLs from "./options.jsx";
import './index.css';

const TorBotArguments = (props) => (
    <select className="dropdown-select" onChange={props.handler}>
        {
            props.args.map((arg) =>
                <option name='argument' key={arg} value={arg}>{arg}</option>
            )
        }
    </select>
);

export const TorBotForm = (props) => {
    const [option, setOption] = useState('Retrieve Mail');
    const [website, setWebsite] = useState('');

    const inputChange = (event) =>
        setWebsite(event.target.value);

    const handleChange = (event) =>
        setOption(event.target.value);


    const handleSubmit = (event) => {
        event.preventDefault();
        event.stopPropagation();
        if (option === 'Retrieve URLs') {
            fetch('http://localhost:8008/LIVE', {
                body: JSON.stringify({
                    option,
                    website,
                }),
                method: 'POST'
            }).then(response => {
                // object has 'websites' property that contains an array of links
                return response.json();
            }).then(data => {
                handleURLs(data);
            }).catch(error => {
                alert(error);
            });
        }
    };

    return (
        <form onSubmit={handleSubmit} id="mainForm">
            <label id='siteFieldTitle'> Website:
                <input onChange={inputChange} id='siteName' type='text' name='website'/>
            </label>
            <br/>
            <label id='optionTitle'> Option:
                <TorBotArguments handler={handleChange} args={props.args}/>
            </label>
            <br/>
            <input id='submitBtn' type="submit"/>
        </form>
    );
};


function handleURLs(data) {
    ReactDOM.render(<DisplayURLs flags={flags} websites={data.websites}/>, document.getElementById('root'));
}

const flags = ['Retrieve Emails', 'Retrieve URLs', 'Retrieve Information'];
ReactDOM.render(<TorBotForm args={flags}/>, document.getElementById('root'));

