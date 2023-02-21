import shutil
import os
import json

# source = "dataset/etherdelta_3-9/stake"
# target = "dataset/etherdelta_10-14/stake"

# for file in os.listdir(source):
#     # if file[:2] == "13" or file[:2] == "14":
#     if file[:2] in ["10","11","12","13","14"]:
#         # print(file)
#         shutil.move(f"{source}/{file}",f"{target}/{file}")

# source = "fuzz"
# for dapp in os.listdir(source):
#     shutil.rmtree(f"{source}/{dapp}/msg/input")
#     shutil.rmtree(f"{source}/{dapp}/msg/output")
    # os.mkdir(f"{source}/{dapp}/msg")
    # shutil.move(f"{source}/{dapp}/proxy", f"{source}/{dapp}/msg/proxy")
    # shutil.move(f"{source}/{dapp}/stake", f"{source}/{dapp}/msg/stake")


# source = "abi_"
# for abi in os.listdir(source):
#     os.rename(f"{source}/{abi}",f"{source}/{abi.lower()}")

# tx_list = os.listdir("dataset/etherdelta_3-9/proxy")
# tx_list.sort()
# print(tx_list[0:10])

# with open("temp_txt.txt",'r') as f:
#     lines = f.readlines()

# with open("token.txt","r") as f:
#     tokenLines = f.readlines()

# with open("tokenInfo.csv",'w') as f:
#     for i, line in enumerate(tokenLines):
#         line = line.split('\n')[0]
#         f.write(f"{','.join(line.split(' '))},{lines[i].split(' ')[2]}")

# source = "seed_and_env"
# for d in os.listdir(source):
#     shutil.rmtree(f"{source}/{d}/statedb/input")

# print(os.listdir("seed_and_env/aave_v2/msg/proxy")[-1])

# with open("tokenInfo.csv",'r') as f:
#     lines = f.readlines()

# with open("tokenToWeth.csv","w") as f:
#     for i,line in enumerate(lines):
#         line = line.split('\n')[0]
#         if i == 0:
#             f.write(f"{line},decimals\n")
#         else:
#             f.write(f"{line},1\n")

# source = "dataset"
# block = 12000000
# for dapp in os.listdir(source):
#     print(dapp, len(os.listdir(f"{source}/{dapp}/proxy")))

# lines = 0
# target = "fuzz"
# for file in os.listdir(target):
#     if file.split(".")[-1] != "go":
#         continue
#     print(file)
#     with open(f"{target}/{file}","r") as f:
#         lines += len(f.readlines())

# print(lines)

# with open("defi.csv","r") as f:
#     lines = f.readlines()
#     for line in lines:
#         line = line.split("\n")[0]
#         content = line.split(",")
#         name = content[0].strip()
#         proxy = content[4].strip().lower()
#         # if not os.path.exists(f"defi_apps/{name}"):
#         #     os.mkdir(f"defi_apps/{name}")
#         # if os.path.exists(f"abi/{proxy}.json"):
#         #     shutil.copy(f"abi/{proxy}.json",f"defi_apps/{name}/{proxy}.json")
#         # else:
#         #     print(name,proxy)
#         print(name,proxy)

# tokenInfo = dict()
# with open("tokenToWeth.csv", "r") as f:
#     lines = [l.split('\n')[0] for l in f.readlines()[1:]]
#     for line in lines:
#         # symbol, tokenAddress, swapAddress,decimals
#         symbol, tokenAddress, swapAddress, decimals = [x.strip() for x in line.split(",")]
#         tokenInfo[symbol] = {
#             "tokenAddress": tokenAddress,
#             "decimals": decimals,
#         }

# with open("temp/token.csv","r") as f:
#     lines = [l.split('\n')[0] for l in f.readlines()[:]]
#     for line in lines:
#         symbol, tokenAddress = [x.strip() for x in line.split(",")]
#         tokenInfo[symbol] = {
#             "tokenAddress": tokenAddress,
#             "decimals": 0,
#         }

# tokenSwap_set = set()
# with open("tokenSwap/tokenToToken.csv", "r") as f:
#     lines = f.readlines()
#     print(len(lines))
#     for line in lines:
#         atoken, btoken, swapAddress = line.split("\n")[0].split(" ")
#         # print(atoken, btoken, address)
#         token_list = [atoken, btoken]
#         # token_list.sort()
#         if swapAddress != "0x0000000000000000000000000000000000000000":
#             tokenSwap_set.add(f"{atoken},{tokenInfo[atoken]['tokenAddress']},{tokenInfo[atoken]['decimals']},{btoken},{swapAddress}")

# print(len(tokenSwap_set))
# with open("tokenSwap/tokenSwap.csv", "w") as f1:
#     for item in tokenSwap_set:
#         f1.write(f"{item}\n")

token_dict = dict()
with open("tokenSwap.csv", "r") as f:
    lines = f.readlines()[1:]
    for line in lines:
        line = [x.strip().lower() for x in line.split(",")]
        token_dict[line[0]] = {
            "address":line[1],
            "decimals":int(line[2])
        }

with open("token.json","w") as f:
    json.dump(token_dict,f)