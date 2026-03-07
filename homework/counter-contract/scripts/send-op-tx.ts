import hre from "hardhat";

async function main() {
  // 连接 hardhat.config.ts 中配置的 hardhatOp 网络
  // ethers 由 hardhat-ethers 插件在运行时注入，需类型断言
  const connection = await hre.network.connect("hardhatOp") as unknown as { ethers: { getSigners(): Promise<Array<{ address: string; sendTransaction(tx: object): Promise<{ wait(): Promise<unknown> }> }>> } };
  const { ethers } = connection;

  console.log("Sending transaction using the OP chain type");
  const [sender] = await ethers.getSigners();
  console.log("Sending 1 wei from", sender.address, "to itself");

  console.log("Sending L2 transaction");
  const tx = await sender.sendTransaction({
    to: sender.address,
    value: 1n,
  });

  await tx.wait();

  console.log("Transaction sent successfully");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
