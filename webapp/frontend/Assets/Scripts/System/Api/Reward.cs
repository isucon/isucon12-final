using System;
using System.Threading.Tasks;

namespace Network
{
    [Serializable]
    public class RewardRequest : CommonRequest
    {
    }

    [Serializable]
    public class RewardResponse : CommonResponse
    {
    }
    
    public class RewardApi : ApiBase<RewardRequest, RewardResponse>
    {
        public override string Path { get; } = $"/user/{GameManager.userData.user.id}/reward";

        public RewardApi()
        {
            RequestData = new RewardRequest();
        }
    }

    public partial class ApiClient
    {
        public async Task<RewardResponse> RewardAsync()
        {
            var api = new RewardApi();
            var res = await Post(api);
            return res;
        }
    }
}
