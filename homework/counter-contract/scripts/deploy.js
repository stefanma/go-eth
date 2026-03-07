import hre from "hardhat";

async function main() {
    const { ethers } = await hre.network.connect();
    const counter = await ethers.deployContract("Counter", [0]);

    await counter.waitForDeployment();

    console.log("Counter deployed to:", await counter.getAddress());
}

// 运行脚本
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });