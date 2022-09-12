using System;
using System.Threading.Tasks;
using Data;
using Network;

public static class GameManager
{
    public static ApiClient apiClient = new("");

    public static UserData userData = new();
    
    public static async Task CreateUserAsync()
    {
        var res = await apiClient.CreateUserAsync(Guid.NewGuid().ToString());
        UpdateCommonResources(res);
        userData.user.id = res.userId;
        userData.user.sessionId = res.sessionId;
    }
    
    public static async Task HomeAsync()
    {
        var res = await apiClient.HomeAsync();
        // homeはレスポンスが特殊なのでUpdateCommonResourcesを使わない
        userData.user = res.user;
        userData.isuCoin.totalPerSec = res.totalAmountPerSec;
        userData.isuCoin.pastTime = res.pastTime;
        userData.isuCoin.refreshTime = DateTimeOffset.Now.ToUnixTimeSeconds();
        
        if (res.deck.id != 0)
        {
            // JsonUtilityの仕様上nullが来ない
            userData.userDeck = res.deck;
        }
    }


    private static void UpdateCommonResources(CommonResponse common)
    {
        if (common?.updatedResources == null)
        {
            return;
        }

        userData.user = common.updatedResources.user;

        if (common.updatedResources.userLoginBonuses != null)
        {
            foreach (var loginBonus in common.updatedResources.userLoginBonuses)
            {
                userData.loginBonuses.Add(loginBonus);
            }
        }

        if (common.updatedResources.userPresents != null)
        {
            foreach (var present in common.updatedResources.userPresents)
            {
                userData.presents.Add(present);
            }
        }
    }
}
