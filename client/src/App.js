import React, { Component } from 'react';
import { hot } from 'react-hot-loader';
import { TextField, Select, InputLabel, MenuItem, Button } from '@material-ui/core';
import { styled } from '@material-ui/styles';
import axios from 'axios';
import './App.css';

class Selector extends Component {
    constructor(props) {
        super(props);
        this.state = {
            list: props.list,
            label: props.label,
            itemIndex: props.itemIndex
        };
        this.handleChange = this.handleChange.bind(this);
    }

    handleChange(event) {
        this.setState({
            itemIndex: event.target.value
        });
        this.props.onChange(this.state.list[event.target.value]);
    }

    render() {
        return (
            <div>
                <InputLabel id="torbotOptions">{this.state.label}</InputLabel>
                <Select onChange={this.handleChange} id="torbotOptions" value={this.state.itemIndex}>
                    {this.state.list.map((element, index) => <MenuItem key={index} value={index}>{element}</MenuItem>)}
                </Select>
            </div>
        );
    }
}

const MainTextField = styled(TextField)({
    'padding-bottom': '10%'
});

const getLinks = (url) => {
    const urlParam = encodeURIComponent(url);
    return axios.get(`http://localhost:3050?url=${urlParam}`);
};

class App extends Component {
    constructor(props) {
        super(props);
        this.state = {
            url: '',
            selected: 'Get Links', 
            options: ['Get Links', 'Analyze']
        };
        this.handleSubmit = this.handleSubmit.bind(this);
        this.handleTextChange = this.handleTextChange.bind(this);
        this.handleOptionChange = this.handleOptionChange.bind(this);
    }

    handleSubmit(event) {
        switch (this.state.selected) {
            case 'Get Links':
                getLinks(this.state.url)
                    .then(response => console.log(response.data))
                    .catch(err => console.error(err));
                break;
            case 'Analyze':
                console.log('Analyzing links');
                break;
        }
    }

    handleOptionChange(newOption) {
        this.setState({
            selected: newOption
        });
    }

    handleTextChange(event) {
        this.setState({
            url: event.target.value
        });
    }

    render() {
        return (
            <div style={{
                position: 'absolute', left: '50%', top: '50%',
                transform: 'translate(-50%, -50%)'
            }} className="App">
                <MainTextField onChange={this.handleTextChange} label="URL" color="primary"/>
                <br/>
                <Selector onChange={this.handleOptionChange} list={this.state.options} label="Options" itemIndex={0}></Selector>
                <br/>
                <Button onClick={this.handleSubmit}>submit</Button>
            </div>
        );
    }
}

export default hot(module)(App);