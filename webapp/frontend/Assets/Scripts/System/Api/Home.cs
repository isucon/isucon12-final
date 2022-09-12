using System;
using System.Threading.Tasks;
using Data;
using UnityEngine.Networking;

namespace Network
{
    [Serializable]
    public class HomeResponse : CommonResponse
    {
        public User user;
        public UserDeck deck;
        public int totalAmountPerSec;
        public long pastTime;
        public long now;
    }
    
    [Serializable]
    public class HomeApi : ApiBase<CommonRequest, HomeResponse>
    {
        public override string Path { get; } = $"/user/{GameManager.userData.user.id}/home";
        public override string Method => UnityWebRequest.kHttpVerbGET;
    }

    public partial class ApiClient
    {
        public async Task<HomeResponse> HomeAsync()
        {
            var api = new HomeApi();
            var res = await Post(api);
            return res;
        }
    }
}
