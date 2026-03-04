// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

contract Counter {
    uint256 private _count;
    address public owner;

    event CountChanged(address indexed caller, uint256 newValue, string action);
    
    event OwnershipTransferred(
        address indexed previousOwner,
        address indexed newOwner
    );

    modifier onlyOwner() {
        require(msg.sender == owner, "Counter: caller is not the owner");
        _;
    }

    constructor(uint256 initialValue) {
        owner = msg.sender;
        _count = initialValue;
        emit CountChanged(msg.sender, initialValue, "initialize");
    }

    function increment() external {
        _count += 1;
        emit CountChanged(msg.sender, _count, "increment");
    }

    function incrementBy(uint256 step) external {
        require(step > 0, "Counter: step must be > 0");
        _count += step;
        emit CountChanged(msg.sender, _count, "incrementBy");
    }

    function decrement() external {
        require(_count > 0, "Counter: count is already zero");
        _count -= 1;
        emit CountChanged(msg.sender, _count, "decrement");
    }

    function reset() external onlyOwner {
        _count = 0;
        emit CountChanged(msg.sender, 0, "reset");
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Counter: new owner is zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    function getCount() external view returns (uint256) {
        return _count;
    }

    function version() external pure returns (string memory) {
        return "1.0.0";
    }
}
